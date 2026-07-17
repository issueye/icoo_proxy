//go:build windows

package main

import (
	_ "embed"
	"sync"

	"github.com/getlantern/systray"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed build/windows/icon.ico
var trayIcon []byte

type trayController struct {
	once sync.Once
}

func (a *App) startTray() {
	if a.tray == nil {
		a.tray = &trayController{}
	}
	a.tray.once.Do(func() {
		go systray.Run(a.onTrayReady, nil)
	})
}

func (a *App) onTrayReady() {
	if len(trayIcon) > 0 {
		systray.SetIcon(trayIcon)
	}
	systray.SetTitle("icoo_desktop")
	systray.SetTooltip("icoo_desktop - icoo_llm_bridge 管理客户端")

	showItem := systray.AddMenuItem("显示主窗口", "恢复并显示主窗口")
	systray.AddSeparator()
	startItem := systray.AddMenuItem("启动 Bridge", "启动 icoo_llm_bridge")
	stopItem := systray.AddMenuItem("停止 Bridge", "停止 icoo_llm_bridge")
	systray.AddSeparator()
	quitItem := systray.AddMenuItem("退出", "退出应用")

	go func() {
		for {
			select {
			case <-showItem.ClickedCh:
				a.showMainWindow()
			case <-startItem.ClickedCh:
				_ = a.StartServer()
			case <-stopItem.ClickedCh:
				_ = a.StopServer()
			case <-quitItem.ClickedCh:
				a.quitFromTray()
				return
			}
		}
	}()
}

func (a *App) showMainWindow() {
	if a == nil || a.ctx == nil {
		return
	}
	wailsruntime.WindowUnminimise(a.ctx)
	wailsruntime.Show(a.ctx)
	wailsruntime.WindowShow(a.ctx)
	wailsruntime.WindowCenter(a.ctx)
	wailsruntime.WindowSetAlwaysOnTop(a.ctx, true)
	wailsruntime.WindowSetAlwaysOnTop(a.ctx, false)
}

func (a *App) quitFromTray() {
	// Stop bridge (+ plugins) before tearing down the UI so orphans are not left
	// if OnShutdown is delayed or skipped.
	if a != nil {
		_ = a.StopServer()
	}
	if a == nil || a.ctx == nil {
		systray.Quit()
		return
	}
	wailsruntime.Quit(a.ctx)
}
