package main

import (
	"context"
	"embed"
	"fmt"
	"log"

	"my-web-mark/app"
	"my-web-mark/internal/appstate"
	"my-web-mark/internal/favicon"
	"my-web-mark/internal/store"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
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
	favCache := favicon.NewCache()
	services := app.NewServices(st, favCache)

	// 启动备份（异步，不阻塞窗口）：保留最近 5 份
	go func() {
		if err := st.Backup(5); err != nil {
			log.Printf("数据库备份失败: %v", err)
		}
	}()

	if err := wails.Run(&options.App{
		Title:     "My Web Mark · 书签与账号管理",
		Width:     1280,
		Height:    800,
		MinWidth:  1024,
		MinHeight: 640,
		// 无边框：隐藏系统标题栏，由前端自绘标题栏（--wails-draggable 拖拽区）
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
			// 兜底服务本地 favicon 缓存（/favicons/<name>）
			Handler: favicon.Handler(favCache),
		},
		BackgroundColour: &options.RGBA{R: 15, G: 18, B: 28, A: 1},
		OnStartup: func(ctx context.Context) {
			// 保留上下文以便后续使用（窗口控制、文件对话框、事件推送）
			appCtx = ctx
			services.SetContext(ctx)
			applyWindowState(ctx)
		},
		OnBeforeClose: func(ctx context.Context) bool {
			saveWindowState(ctx)
			return false
		},
		OnShutdown: func(ctx context.Context) {
			services.Shutdown()
		},
		Bind: services.BindTargets(),
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "my-web-mark-7c3f2a91-b8e4-4c2d",
			// 二次启动：唤起已有实例（取消最小化），不新开窗口
			OnSecondInstanceLaunch: func(data options.SecondInstanceData) {
				if appCtx != nil {
					wailsruntime.WindowUnminimise(appCtx)
				}
			},
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			Theme:                windows.Dark,
		},
	}); err != nil {
		fmt.Println("启动失败:", err)
	}
}

// appCtx 保存 Wails 运行时上下文（窗口控制、事件推送）。
var appCtx context.Context

// applyWindowState 启动时恢复上次窗口状态（多屏降级校验见 appstate.Sanitize）。
func applyWindowState(ctx context.Context) {
	st, err := appstate.Load()
	if err != nil || st == nil {
		return
	}
	if st.Maximised {
		// 上次关闭时为最大化：直接最大化，还原尺寸由系统保留
		wailsruntime.WindowMaximise(ctx)
		return
	}
	fixed := appstate.Sanitize(*st, appstate.VirtualScreen(), appstate.PrimaryScreen(), 1024, 640, 1280, 800)
	wailsruntime.WindowSetSize(ctx, fixed.W, fixed.H)
	wailsruntime.WindowSetPosition(ctx, fixed.X, fixed.Y)
}

// saveWindowState 关闭前快照窗口位置/大小/最大化状态。
func saveWindowState(ctx context.Context) {
	maxed := wailsruntime.WindowIsMaximised(ctx)
	if maxed {
		// 最大化状态下 rect 是最大化尺寸，还原尺寸系统自持：仅记录标记
		if err := appstate.Save(appstate.WindowState{Maximised: true}); err != nil {
			log.Printf("保存窗口状态失败: %v", err)
		}
		return
	}
	x, y := wailsruntime.WindowGetPosition(ctx)
	w, h := wailsruntime.WindowGetSize(ctx)
	if err := appstate.Save(appstate.WindowState{X: x, Y: y, W: w, H: h}); err != nil {
		log.Printf("保存窗口状态失败: %v", err)
	}
}
