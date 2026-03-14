//go:build !wails

package main

import (
	entry "pause/internal/entry/desktop"
	"pause/internal/logx"
)

func main() {
	if err := entry.RunHeadless(""); err != nil {
		logx.Fatalf("failed to init app: %v", err)
	}
}
