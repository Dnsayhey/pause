//go:build wails

package app

import (
	"context"

	"pause/internal/desktop"
	"pause/internal/logx"
)

// BeforeClose intercepts window close requests.
// We hide only the main window and keep tray process alive,
// unless an explicit quit flow requested app termination.
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	if a == nil {
		return false
	}

	if a.quitRequested.Swap(false) {
		logx.Infof("window.before_close allow=true reason=quit_requested")
		return false
	}

	// Allow runtime-driven shutdown (for example Ctrl+C in dev mode).
	if ctx != nil {
		select {
		case <-ctx.Done():
			logx.Infof("window.before_close allow=true reason=context_done")
			return false
		default:
		}
	}

	desktop.HideMainWindowForClose(ctx)
	logx.Infof("window.before_close prevent=true action=hide_main_window")
	return true
}
