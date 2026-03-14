//go:build wails

package main

import (
	"embed"
	"log"

	entry "pause/internal/entry/desktop"
)

//go:embed all:frontend/dist
var bundledAssets embed.FS

func main() {
	if err := entry.RunWailsFromEmbedded("", bundledAssets, "frontend/dist"); err != nil {
		log.Fatal(err)
	}
}
