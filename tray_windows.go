//go:build windows

package main

import (
	"sync"

	"github.com/getlantern/systray"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type trayController struct {
	icon []byte
	once sync.Once
}

func (a *App) registerTray(icon []byte) {
	a.tray = &trayController{icon: icon}
}

func (a *App) startTray() {
	if a.tray == nil {
		return
	}
	a.tray.once.Do(func() {
		systray.Register(a.onTrayReady, nil)
	})
}

func (a *App) onTrayReady() {
	if len(a.tray.icon) > 0 {
		systray.SetIcon(a.tray.icon)
	}
	systray.SetTitle("icoo_proxy")
	systray.SetTooltip("icoo_proxy 本地 AI 协议转换网关")

	showItem := systray.AddMenuItem("显示主窗口", "恢复并显示主窗口")
	systray.AddSeparator()
	quitItem := systray.AddMenuItem("退出", "退出应用")

	go func() {
		for {
			select {
			case <-showItem.ClickedCh:
				a.showMainWindow()
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
	if a == nil || a.ctx == nil {
		systray.Quit()
		return
	}
	wailsruntime.Quit(a.ctx)
}
