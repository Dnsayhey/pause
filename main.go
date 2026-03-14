//go:build !wails

package main

import (
	"log"

	entry "pause/internal/entry/desktop"
)

func main() {
	if err := entry.RunHeadless(""); err != nil {
		log.Fatalf("failed to init app: %v", err)
	}
}
