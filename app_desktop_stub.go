//go:build !wails

package main

import "context"

type noopDesktopController struct{}

func newDesktopController() desktopController {
	return noopDesktopController{}
}

func (noopDesktopController) OnStartup(context.Context, *App) {}
