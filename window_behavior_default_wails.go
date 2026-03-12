//go:build wails && !darwin

package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func configureDesktopWindowBehavior() {}

func showMainWindowFromStatusBar(ctx context.Context) {
	runtime.Show(ctx)
	runtime.WindowUnminimise(ctx)
	runtime.WindowShow(ctx)
}

func showMainWindowForOverlay(ctx context.Context) {
	runtime.Show(ctx)
	runtime.WindowUnminimise(ctx)
	runtime.WindowShow(ctx)
}
