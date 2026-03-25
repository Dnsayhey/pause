//go:build !wails

package app

import "context"

type noopDesktopController struct{}

func newDesktopController() desktopController {
	return noopDesktopController{}
}

func (noopDesktopController) OnStartup(context.Context, *App) {}
