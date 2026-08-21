package main

import (
	"context"
	"embed"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"pennypickdesktop/internal/config"
	"pennypickdesktop/internal/database"
	"pennypickdesktop/internal/handler"
	"pennypickdesktop/internal/middleware"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	cfg := config.Load()

	// 桌面版：数据存放到用户配置目录，避免依赖工作目录
	dataDir, err := os.UserConfigDir()
	if err != nil {
		dataDir = "."
	}
	dataDir = filepath.Join(dataDir, "PennyPick")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	// 日志写入文件，便于排查桌面端问题
	if lf, err := os.OpenFile(filepath.Join(dataDir, "pennypick.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		log.SetOutput(io.MultiWriter(os.Stderr, lf))
		defer lf.Close()
	}
	log.Printf("桌面版启动，数据目录: %s", dataDir)

	cfg.DatabaseURL = "sqlite:///" + filepath.Join(dataDir, "pennypick.db")

	db, err := database.Open(cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate database: %v", err)
	}
	database.InitAdmin(db, cfg)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), middleware.CORS())
	auth := middleware.NewAuth(cfg, db)
	h := handler.New(db, cfg, auth)
	h.RegisterRoutes(r)

	// 随机端口 + 仅本机监听，避免端口冲突
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	apiURL := "http://127.0.0.1:" + strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)

	srv := &http.Server{Handler: r}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("http server: %v", err)
		}
	}()

	log.Printf("本地 API 地址: %s", apiURL)
	app := NewApp(apiURL, db, auth)
	err = wails.Run(&options.App{
		Title:     "拾财 PennyPick",
		Width:     1280,
		Height:    800,
		MinWidth:  1024,
		MinHeight: 700,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 246, G: 247, B: 250, A: 1},
		OnStartup:        app.startup,
		OnShutdown: func(ctx context.Context) {
			ctx2, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = srv.Shutdown(ctx2)
		},
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatalf("run wails: %v", err)
	}
}
