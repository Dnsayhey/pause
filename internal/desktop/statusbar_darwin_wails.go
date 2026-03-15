//go:build darwin && wails

package desktop

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Foundation

#include <stdlib.h>

void PauseStatusBarInit(void);
void PauseStatusBarUpdate(const char *status, const char *countdown, const char *title, int paused, double progress, const char *remindersPayload);
void PauseStatusBarSetLocaleStrings(
	const char *popoverTitle,
	const char *breakNowButton,
	const char *pauseButton,
	const char *resumeButton,
	const char *openButton,
	const char *aboutMenuItem,
	const char *quitMenuItem,
	const char *moreButtonTip,
	const char *tooltip
);
void PauseStatusBarDestroy(void);
*/
import "C"

import (
	"sync"
	"unsafe"

	_ "pause/internal/desktop/macbridge"
)

type darwinStatusBarController struct{}

var (
	statusBarCallbackMu sync.RWMutex
	statusBarCallback   func(actionID int)
)

func NewStatusBarController() StatusBarController {
	return darwinStatusBarController{}
}

func (darwinStatusBarController) Init(onAction func(actionID int)) {
	statusBarCallbackMu.Lock()
	statusBarCallback = onAction
	statusBarCallbackMu.Unlock()
	C.PauseStatusBarInit()
}

func (darwinStatusBarController) Update(status, countdown, title string, paused bool, progress float64, remindersPayload string) {
	cStatus := C.CString(status)
	cCountdown := C.CString(countdown)
	cTitle := C.CString(title)
	cReminders := C.CString(remindersPayload)
	cPaused := C.int(0)
	if paused {
		cPaused = 1
	}
	defer C.free(unsafe.Pointer(cStatus))
	defer C.free(unsafe.Pointer(cCountdown))
	defer C.free(unsafe.Pointer(cTitle))
	defer C.free(unsafe.Pointer(cReminders))
	C.PauseStatusBarUpdate(cStatus, cCountdown, cTitle, cPaused, C.double(progress), cReminders)
}

func (darwinStatusBarController) SetLocale(strings StatusBarLocaleStrings) {
	cPopoverTitle := C.CString(strings.PopoverTitle)
	cBreakNowButton := C.CString(strings.BreakNowButton)
	cPauseButton := C.CString(strings.PauseButton)
	cResumeButton := C.CString(strings.ResumeButton)
	cOpenButton := C.CString(strings.OpenAppButton)
	cAboutMenuItem := C.CString(strings.AboutMenuItem)
	cQuitMenuItem := C.CString(strings.QuitMenuItem)
	cMoreButtonTip := C.CString(strings.MoreButtonTip)
	cTooltip := C.CString(strings.Tooltip)
	defer C.free(unsafe.Pointer(cPopoverTitle))
	defer C.free(unsafe.Pointer(cBreakNowButton))
	defer C.free(unsafe.Pointer(cPauseButton))
	defer C.free(unsafe.Pointer(cResumeButton))
	defer C.free(unsafe.Pointer(cOpenButton))
	defer C.free(unsafe.Pointer(cAboutMenuItem))
	defer C.free(unsafe.Pointer(cQuitMenuItem))
	defer C.free(unsafe.Pointer(cMoreButtonTip))
	defer C.free(unsafe.Pointer(cTooltip))

	C.PauseStatusBarSetLocaleStrings(
		cPopoverTitle,
		cBreakNowButton,
		cPauseButton,
		cResumeButton,
		cOpenButton,
		cAboutMenuItem,
		cQuitMenuItem,
		cMoreButtonTip,
		cTooltip,
	)
}

func (darwinStatusBarController) Destroy() {
	statusBarCallbackMu.Lock()
	statusBarCallback = nil
	statusBarCallbackMu.Unlock()
	C.PauseStatusBarDestroy()
}

//export statusBarMenuCallbackGo
func statusBarMenuCallbackGo(actionID C.int) {
	action := int(actionID)
	statusBarCallbackMu.RLock()
	cb := statusBarCallback
	statusBarCallbackMu.RUnlock()
	if cb != nil {
		cb(action)
	}
}
