//go:build wails && !darwin

package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"pause/internal/diag"
)

func configureDesktopWindowBehavior() {}

func showMainWindowFromStatusBar(ctx context.Context) {
	diag.Logf("window.show activate source=status_bar_open")
	runtime.Show(ctx)
	runtime.WindowUnminimise(ctx)
	runtime.WindowShow(ctx)
}

func showMainWindowForOverlay(ctx context.Context) {
	diag.Logf("window.show activate source=overlay_fallback")
	runtime.Show(ctx)
	runtime.WindowUnminimise(ctx)
	runtime.WindowShow(ctx)
}

func hideMainWindowForOverlay(_ context.Context) {
	diag.Logf("window.hide source=overlay_native")
}
