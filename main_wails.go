//go:build wails

package main

import (
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

func main() {
	app, err := NewApp("")
	if err != nil {
		log.Fatal(err)
	}

	assets := os.DirFS("frontend/dist")

	if err := wails.Run(&options.App{
		Title:             "Pause",
		Width:             920,
		Height:            580,
		MinWidth:          900,
		MinHeight:         560,
		StartHidden:       true,
		HideWindowOnClose: true,
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHidden(),
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "com.pause.app.single-instance",
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.Startup,
		Bind: []interface{}{
			app,
		},
	}); err != nil {
		log.Fatal(err)
	}
}
