//go:build wails

package main

import (
	"embed"

	entry "pause/internal/entry/desktop"
	"pause/internal/logx"
)

//go:embed all:frontend/dist
var bundledAssets embed.FS

func main() {
	if err := entry.RunWailsFromEmbedded("", bundledAssets, "frontend/dist"); err != nil {
		logx.Fatalf("failed to init app: %v", err)
	}
}
