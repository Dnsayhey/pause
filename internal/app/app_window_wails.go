//go:build wails

package app

import (
	"pause/internal/desktop"
	"pause/internal/logx"
)

// CloseWindow hides the main window while keeping tray/background runtime alive.
func (a *App) CloseWindow() {
	if a == nil {
		return
	}
	if a.ctx == nil {
		logx.Warnf("app.close_window skipped reason=missing_context")
		return
	}
	logx.Infof("app.close_window action=hide_main_window")
	desktop.HideMainWindowForClose(a.ctx)
}
