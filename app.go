package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"

	"pennypickdesktop/internal/config"
	"pennypickdesktop/internal/database"
	"pennypickdesktop/internal/handler"
	"pennypickdesktop/internal/middleware"
	"pennypickdesktop/internal/securestore"
)

// App Wails 应用绑定对象，向前端暴露桌面能力。
//
// 数据库在解锁（UnlockDatabase）后才打开并启动内嵌 API 服务。
// 首次解锁后主密码用 Windows DPAPI 加密保存（绑定当前账户），
// 之后启动自动解锁，无需再次输入；业务登录仍使用用户名/密码。
type App struct {
	ctx        context.Context
	cfg        *config.Config
	secretPath string // DPAPI 密钥保存文件
	handle     *database.Handle
	db         *gorm.DB
	auth       *middleware.Auth
	srv        *http.Server
	apiBase    string
	stopFlush  chan struct{}
}

func NewApp(cfg *config.Config) *App {
	secretPath := ""
	if p := strings.TrimPrefix(cfg.DatabaseURL, "sqlite:///"); p != "" {
		secretPath = filepath.Join(filepath.Dir(p), "dbkey.bin")
	}
	return &App{cfg: cfg, secretPath: secretPath}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetDatabaseStatus 返回数据库状态（前端据此决定是否显示解锁界面）：
//
//	"ready"      已解锁（本次启动已用保存的密钥自动解锁）
//	"encrypted"  已加密，请输入主密码解锁
//	"plain"      存在明文数据库，请设置主密码（首次启用加密）
//	"new"        全新数据库，请设置主密码
func (a *App) GetDatabaseStatus() (string, error) {
	if a.db != nil {
		return "ready", nil
	}
	dbPath := strings.TrimPrefix(a.cfg.DatabaseURL, "sqlite:///")
	st, err := database.Status(dbPath)
	if err != nil {
		return "", err
	}
	// 已加密且本机保存有密钥：自动解锁，免去每次输入
	if st == "encrypted" && a.tryAutoUnlock() {
		return "ready", nil
	}
	return st, nil
}

// tryAutoUnlock 尝试用本机 DPAPI 保存的密钥自动解锁数据库。
func (a *App) tryAutoUnlock() bool {
	secret, ok := securestore.Load(a.secretPath)
	if !ok {
		return false
	}
	if _, err := a.unlock(secret); err != nil {
		log.Printf("自动解锁失败（保存的密钥可能已失效，请手动输入主密码）: %v", err)
		return false
	}
	log.Printf("已用本机保存的密钥自动解锁数据库")
	return true
}

// UnlockDatabase 输入主密码：解密/迁移并打开数据库，启动本地 API 服务。
// 首次设置或重新输入成功后，会把主密码加密保存到本机，之后启动自动解锁。
// 返回本地 API 基础地址（含 /api 前缀），供前端注入 axios 使用。
func (a *App) UnlockDatabase(passphrase string) (string, error) {
	if a.db != nil {
		return a.apiBase, nil // 已解锁
	}
	if strings.TrimSpace(passphrase) == "" {
		return "", fmt.Errorf("主密码不能为空")
	}
	apiBase, err := a.unlock(passphrase)
	if err != nil {
		return "", err
	}
	if a.secretPath != "" {
		if err := securestore.Save(a.secretPath, passphrase); err != nil {
			log.Printf("[warn] 保存主密码到本机失败（不影响本次使用）: %v", err)
		}
	}
	return apiBase, nil
}

// unlock 核心解锁流程：解密/迁移数据库 → 启动本地 API 服务。
func (a *App) unlock(passphrase string) (string, error) {
	a.cfg.DatabasePass = passphrase
	h, err := database.Open(a.cfg)
	if err != nil {
		return "", err
	}
	if err := database.Migrate(h.DB()); err != nil {
		_ = h.Close()
		return "", fmt.Errorf("初始化数据库失败: %v", err)
	}
	database.InitAdmin(h.DB(), a.cfg)

	a.handle = h
	a.db = h.DB()
	a.auth = middleware.NewAuth(a.cfg, a.db)
	hh := handler.New(a.db, a.cfg, a.auth)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), middleware.CORS())
	hh.RegisterRoutes(r)

	// 随机端口 + 仅本机监听，避免端口冲突
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = h.Close()
		return "", fmt.Errorf("监听失败: %v", err)
	}
	apiURL := "http://127.0.0.1:" + strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	a.srv = &http.Server{Handler: r}
	go func() {
		if err := a.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("http server: %v", err)
		}
	}()

	// 加密模式每 5 分钟回写一次 .enc，缩小崩溃丢失窗口
	a.stopFlush = make(chan struct{})
	go h.AutoFlush(5*time.Minute, a.stopFlush)

	a.apiBase = apiURL + "/api"
	log.Printf("数据库已解锁，本地 API 地址: %s", apiURL)
	return a.apiBase, nil
}

// GetAPIBaseURL 返回本地 API 基础地址（前端 axios 使用）。
func (a *App) GetAPIBaseURL() string {
	return a.apiBase
}

// closeDB 应用退出时调用：关闭 API 服务并回写加密数据库。
func (a *App) closeDB() {
	if a.stopFlush != nil {
		close(a.stopFlush)
		a.stopFlush = nil
	}
	if a.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = a.srv.Shutdown(ctx)
		cancel()
		a.srv = nil
	}
	if a.handle != nil {
		if err := a.handle.Close(); err != nil {
			log.Printf("close database: %v", err)
		}
		a.handle = nil
		a.db = nil
	}
}

// ExportPlainDB 导出明文数据库到用户指定位置，用于迁移到其他电脑。
// 仅管理员可用；返回保存的文件路径；用户取消时返回空字符串。
func (a *App) ExportPlainDB(token string) (string, error) {
	if a.handle == nil || a.db == nil || a.auth == nil {
		return "", fmt.Errorf("数据库尚未解锁")
	}
	user, ok := a.auth.ParseUserFromToken(strings.TrimPrefix(token, "Bearer "))
	if !ok {
		return "", fmt.Errorf("登录凭证无效或已过期")
	}
	if user.Username != a.cfg.AdminUsername {
		return "", fmt.Errorf("仅管理员可导出明文数据库")
	}
	if a.ctx == nil {
		return "", fmt.Errorf("窗口尚未就绪")
	}
	saved, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: "pennypick_plain_" + time.Now().Format("2006-01-02_150405") + ".db",
		Title:           "导出明文数据库（迁移用）",
		Filters: []runtime.FileFilter{
			{DisplayName: "SQLite 数据库", Pattern: "*.db"},
		},
	})
	if err != nil {
		return "", err
	}
	if saved == "" {
		return "", nil // 用户取消
	}
	if err := a.handle.ExportPlainDB(saved); err != nil {
		return "", fmt.Errorf("导出明文数据库失败: %v", err)
	}
	return saved, nil
}

// ExportBills 导出账单 CSV：弹出保存对话框并写入文件。
// 返回保存的文件路径；用户取消时返回空字符串。
func (a *App) ExportBills(start, end, typ, token string) (string, error) {
	if a.db == nil || a.auth == nil {
		return "", fmt.Errorf("数据库尚未解锁")
	}
	user, ok := a.auth.ParseUserFromToken(strings.TrimPrefix(token, "Bearer "))
	if !ok {
		return "", fmt.Errorf("登录凭证无效或已过期")
	}
	content, err := handler.BuildBillsCSV(a.db, user.ID, start, end, typ)
	if err != nil {
		return "", fmt.Errorf("生成 CSV 失败: %v", err)
	}
	if a.ctx == nil {
		return "", fmt.Errorf("窗口尚未就绪")
	}
	saved, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: "pennypick_bills_" + time.Now().Format("2006-01-02_150405") + ".csv",
		Title:           "导出账单 CSV",
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV 文件", Pattern: "*.csv"},
		},
	})
	if err != nil {
		return "", err
	}
	if saved == "" {
		return "", nil // 用户取消
	}
	if err := os.WriteFile(saved, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("写入文件失败: %v", err)
	}
	return saved, nil
}
