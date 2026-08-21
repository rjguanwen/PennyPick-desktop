package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"

	"pennypickdesktop/internal/handler"
	"pennypickdesktop/internal/middleware"
)

// App Wails 应用绑定对象，向前端暴露桌面能力。
type App struct {
	ctx    context.Context
	apiURL string
	db     *gorm.DB
	auth   *middleware.Auth
}

func NewApp(apiURL string, db *gorm.DB, auth *middleware.Auth) *App {
	return &App{apiURL: apiURL, db: db, auth: auth}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetAPIBaseURL 返回本地 API 基础地址（前端 axios 使用）。
func (a *App) GetAPIBaseURL() string {
	return a.apiURL + "/api"
}

// ExportBills 导出账单 CSV：弹出保存对话框并写入文件。
// 返回保存的文件路径；用户取消时返回空字符串。
func (a *App) ExportBills(start, end, typ, token string) (string, error) {
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
