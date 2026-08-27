package main

import (
	"context"
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"pennypickdesktop/internal/config"
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

	// 日志写入文件，便于排查桌面端问题。
	// 注意：GUI 程序无控制台时 os.Stderr 无效，io.MultiWriter 会因此中断，
	// 故只写文件，不写 stderr。
	if lf, err := os.OpenFile(filepath.Join(dataDir, "pennypick.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		log.SetOutput(lf)
		defer lf.Close()
	}
	log.Printf("桌面版启动，数据目录: %s", dataDir)

	cfg.DatabaseURL = "sqlite:///" + filepath.Join(dataDir, "pennypick.db")

	// 数据库不在启动时打开：先起窗口，由前端解锁界面输入主密码后通过
	// UnlockDatabase 绑定完成解密/迁移并启动本地 API 服务。
	app := NewApp(cfg)
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
			app.closeDB()
		},
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatalf("run wails: %v", err)
	}
}
