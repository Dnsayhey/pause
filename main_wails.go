//go:build wails

package main

import (
	"embed"
	"io/fs"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var bundledAssets embed.FS

func mustBundledAssets() fs.FS {
	assets, err := fs.Sub(bundledAssets, "frontend/dist")
	if err != nil {
		log.Fatalf("failed to load bundled frontend assets: %v", err)
	}
	return assets
}

func main() {
	app, err := NewApp("")
	if err != nil {
		log.Fatal(err)
	}

	assets := mustBundledAssets()

	if err := wails.Run(&options.App{
		Title:             "Pause",
		Width:             820,
		Height:            520,
		MinWidth:          820,
		MinHeight:         520,
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
