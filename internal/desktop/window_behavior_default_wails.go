//go:build wails && !darwin && !windows

package desktop

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func configureDesktopWindowBehavior() {}

func ShowMainWindowFromStatusBar(ctx context.Context) {
	runtime.Show(ctx)
	runtime.WindowUnminimise(ctx)
	runtime.WindowShow(ctx)
}

func ShowMainWindowForOverlay(ctx context.Context) {
	runtime.Show(ctx)
	runtime.WindowUnminimise(ctx)
	runtime.WindowShow(ctx)
}

func HideMainWindowForClose(ctx context.Context) {
	runtime.WindowHide(ctx)
}

func HideMainWindowForOverlay(ctx context.Context) {
	HideMainWindowForClose(ctx)
}

func ConfigureDesktopWindowBehavior() {
	configureDesktopWindowBehavior()
}
