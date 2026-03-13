//go:build wails

package main

import (
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) Quit() {
	if a == nil {
		return
	}
	a.quitRequested.Store(true)
	if a.ctx == nil {
		return
	}
	runtime.Quit(a.ctx)
}
