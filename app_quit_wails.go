//go:build wails

package main

import "github.com/wailsapp/wails/v2/pkg/runtime"

func (a *App) Quit() {
	if a == nil || a.ctx == nil {
		return
	}
	runtime.Quit(a.ctx)
}
