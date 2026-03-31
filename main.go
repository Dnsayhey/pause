//go:build !wails

package main

import (
	"os"

	entry "pause/internal/entry/desktop"
	"pause/internal/logx"
)

func main() {
	if err := entry.RunHeadless(""); err != nil {
		logx.Errorf("failed to init app: %v", err)
		os.Exit(1)
	}
}
