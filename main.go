package main

import (
	"embed"
	goruntime "runtime"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

const Version = "0.0.1"

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/windows/icon.ico
var trayIcon []byte

func main() {
	app := NewApp()
	if goruntime.GOOS == "windows" {
		app.registerTray(trayIcon)
	}

	err := wails.Run(&options.App{
		Title:             "icoo Proxy",
		Width:             1080,
		Height:            860,
		MinWidth:          900,
		MinHeight:         600,
		MaxWidth:          1080,
		MaxHeight:         860,
		DisableResize:     false,
		Frameless:         true,
		Fullscreen:        false,
		HideWindowOnClose: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 245, G: 245, B: 245, A: 255},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}

	systray.Quit()
}
