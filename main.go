package main

import (
	"embed"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	isAutostart := false
	for _, arg := range os.Args {
		if strings.ToLower(arg) == "--autostart" {
			isAutostart = true
			break
		}
	}

	err := wails.Run(&options.App{
		Title:            "Onion VPN",
		Width:            820,
		Height:           900,
		Assets:           assets,
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 59, A: 1},
		StartHidden:      isAutostart,
		OnStartup:        app.Startup,
		OnDomReady:       app.DomReady,
		OnShutdown:       app.Shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			OnSuspend:            app.MinimizeToTray,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
