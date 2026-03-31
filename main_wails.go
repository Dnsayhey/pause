//go:build wails

package main

import (
	"embed"
	"os"

	entry "pause/internal/entry/desktop"
	"pause/internal/logx"
)

//go:embed all:frontend/dist
var bundledAssets embed.FS

func main() {
	if err := entry.RunWailsFromEmbedded("", bundledAssets, "frontend/dist"); err != nil {
		logx.Errorf("failed to init app: %v", err)
		os.Exit(1)
	}
}
