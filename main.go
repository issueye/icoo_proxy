package main

import (
	"embed"
	"icoo_proxy/internal/services"
	"io/fs"
	"net/http"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

type assetHandler struct {
	assets http.Handler
}

func newAssetHandler() *assetHandler {
	sub, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		panic(err)
	}
	return &assetHandler{
		assets: http.FileServer(http.FS(sub)),
	}
}

func (h *assetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.assets.ServeHTTP(w, r)
}

func main() {
	app := services.NewApp()

	err := wails.Run(&options.App{
		Title:  "icoo_proxy - AI Gateway",
		Width:  1200,
		Height: 890,
		AssetServer: &assetserver.Options{
			Handler: newAssetHandler(),
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.Startup,
		OnShutdown:       app.Shutdown,
		Bind: []any{
			app,
		},
		Frameless: true,
		Windows: &windows.Options{
			DisableFramelessWindowDecorations: true,
			WebviewIsTransparent:              true,
			WindowIsTranslucent:               true,
		},
	})

	if err != nil {
		println("错误:", err.Error())
	}
}
