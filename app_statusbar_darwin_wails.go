//go:build darwin && wails

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Foundation

#include <stdlib.h>

void PauseStatusBarInit(void);
void PauseStatusBarUpdate(const char *status, const char *countdown, const char *title, int paused, double progress);
void PauseStatusBarSetLocaleStrings(
	const char *popoverTitle,
	const char *breakNowButton,
	const char *pauseButton,
	const char *pause30Button,
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
)

const (
	statusBarActionBreakNow   = 1
	statusBarActionPause      = 2
	statusBarActionPause30    = 3
	statusBarActionResume     = 4
	statusBarActionOpenWindow = 5
	statusBarActionQuit       = 6
)

type statusBarController interface {
	Init(onAction func(actionID int))
	Update(status, countdown, title string, paused bool, progress float64)
	SetLocale(strings statusBarLocaleStrings)
	Destroy()
}

type darwinStatusBarController struct{}

var (
	statusBarCallbackMu sync.RWMutex
	statusBarCallback   func(actionID int)
)

func newStatusBarController() statusBarController {
	return darwinStatusBarController{}
}

func (darwinStatusBarController) Init(onAction func(actionID int)) {
	statusBarCallbackMu.Lock()
	statusBarCallback = onAction
	statusBarCallbackMu.Unlock()
	C.PauseStatusBarInit()
}

func (darwinStatusBarController) Update(status, countdown, title string, paused bool, progress float64) {
	cStatus := C.CString(status)
	cCountdown := C.CString(countdown)
	cTitle := C.CString(title)
	cPaused := C.int(0)
	if paused {
		cPaused = 1
	}
	defer C.free(unsafe.Pointer(cStatus))
	defer C.free(unsafe.Pointer(cCountdown))
	defer C.free(unsafe.Pointer(cTitle))
	C.PauseStatusBarUpdate(cStatus, cCountdown, cTitle, cPaused, C.double(progress))
}

func (darwinStatusBarController) SetLocale(strings statusBarLocaleStrings) {
	cPopoverTitle := C.CString(strings.PopoverTitle)
	cBreakNowButton := C.CString(strings.BreakNowButton)
	cPauseButton := C.CString(strings.PauseButton)
	cPause30Button := C.CString(strings.Pause30Button)
	cResumeButton := C.CString(strings.ResumeButton)
	cOpenButton := C.CString(strings.OpenAppButton)
	cAboutMenuItem := C.CString(strings.AboutMenuItem)
	cQuitMenuItem := C.CString(strings.QuitMenuItem)
	cMoreButtonTip := C.CString(strings.MoreButtonTip)
	cTooltip := C.CString(strings.Tooltip)
	defer C.free(unsafe.Pointer(cPopoverTitle))
	defer C.free(unsafe.Pointer(cBreakNowButton))
	defer C.free(unsafe.Pointer(cPauseButton))
	defer C.free(unsafe.Pointer(cPause30Button))
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
		cPause30Button,
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
