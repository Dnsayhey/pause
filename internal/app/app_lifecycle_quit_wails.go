//go:build wails

package app

import (
	"pause/internal/logx"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) Quit() {
	if a == nil {
		return
	}
	logx.Infof("app.quit requested")
	a.quitRequested.Store(true)
	if a.ctx == nil {
		logx.Warnf("app.quit skipped reason=missing_context")
		return
	}
	runtime.Quit(a.ctx)
}
