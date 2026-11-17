package main

import (
	"context"
	"embed"
	"fmt"
	"log"

	"my-web-mark/app"
	"my-web-mark/internal/store"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	dbPath, err := store.DefaultDBPath()
	if err != nil {
		log.Fatalf("无法定位数据库路径: %v", err)
	}
	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	services := app.NewServices(st)

	if err := wails.Run(&options.App{
		Title:     "My Web Mark · 书签与账号管理",
		Width:     1280,
		Height:    800,
		MinWidth:  1024,
		MinHeight: 640,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 18, B: 28, A: 1},
		OnStartup: func(ctx context.Context) {
			// 保留上下文以便后续使用（如窗口控制）
			appCtx = ctx
		},
		OnBeforeClose: func(ctx context.Context) bool {
			return false
		},
		OnShutdown: func(ctx context.Context) {
			services.Shutdown()
		},
		Bind: services.BindTargets(),
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			Theme:                windows.Dark,
		},
	}); err != nil {
		fmt.Println("启动失败:", err)
	}
}

// appCtx 保存 Wails 运行时上下文（预留：窗口控制、事件推送）。
var appCtx context.Context
