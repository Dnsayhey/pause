//go:build wails

package app

import (
	"runtime"
	"strings"

	"pause/internal/logx"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) Quit() {
	if a == nil {
		return
	}
	logx.Infof("app.quit requested source=%s", quitCallSource())
	a.quitRequested.Store(true)
	if a.ctx == nil {
		logx.Warnf("app.quit skipped reason=missing_context")
		return
	}
	wailsruntime.Quit(a.ctx)
}

func quitCallSource() string {
	var pcs [8]uintptr
	count := runtime.Callers(2, pcs[:])
	frames := runtime.CallersFrames(pcs[:count])
	sources := make([]string, 0, 4)
	for {
		frame, more := frames.Next()
		if frame.Function != "" && !strings.Contains(frame.Function, "runtime.") {
			sources = append(sources, frame.Function)
			if len(sources) >= 4 {
				break
			}
		}
		if !more {
			break
		}
	}
	if len(sources) == 0 {
		return "unknown"
	}
	return strings.Join(sources, " <- ")
}
