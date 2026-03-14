//go:build darwin && wails

package desktop

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Foundation

void PauseConfigureDesktopWindowBehavior(void);
void PauseShowMainWindowNoActivate(void);
void PauseShowMainWindowActivate(void);
void PauseHideMainWindow(void);
*/
import "C"

import (
	"context"

	_ "pause/internal/desktop/macbridge"
)

func configureDesktopWindowBehavior() {
	C.PauseConfigureDesktopWindowBehavior()
}

func ShowMainWindowFromStatusBar(_ context.Context) {
	C.PauseShowMainWindowActivate()
}

func ShowMainWindowForOverlay(_ context.Context) {
	C.PauseShowMainWindowNoActivate()
}

func HideMainWindowForClose(_ context.Context) {
	C.PauseHideMainWindow()
}

func HideMainWindowForOverlay(ctx context.Context) {
	HideMainWindowForClose(ctx)
}

func ConfigureDesktopWindowBehavior() {
	configureDesktopWindowBehavior()
}
