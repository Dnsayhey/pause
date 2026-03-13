//go:build darwin && wails

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Foundation

#include <stdlib.h>

void PauseBreakOverlayInit(void);
int PauseBreakOverlayShow(int allowSkip, const char *skipButtonTitle, const char *countdownText, const char *theme);
void PauseBreakOverlayHide(void);
void PauseBreakOverlayDestroy(void);
*/
import "C"

import (
	"os"
	"strings"
	"sync"
	"unsafe"
)

type darwinBreakOverlayController struct{}

var (
	overlayCallbackMu sync.RWMutex
	overlayCallback   func()
)

func newBreakOverlayController() breakOverlayController {
	return darwinBreakOverlayController{}
}

func (darwinBreakOverlayController) Init(onSkip func()) {
	overlayCallbackMu.Lock()
	overlayCallback = onSkip
	overlayCallbackMu.Unlock()
	C.PauseBreakOverlayInit()
}

func (darwinBreakOverlayController) Show(allowSkip bool, skipButtonTitle string, countdownText string, theme string) bool {
	if shouldForceOverlayShowFailForDebug() {
		return false
	}

	cTitle := C.CString(skipButtonTitle)
	cCountdown := C.CString(countdownText)
	cTheme := C.CString(theme)
	defer C.free(unsafe.Pointer(cTitle))
	defer C.free(unsafe.Pointer(cCountdown))
	defer C.free(unsafe.Pointer(cTheme))

	cAllowSkip := C.int(0)
	if allowSkip {
		cAllowSkip = 1
	}
	return C.PauseBreakOverlayShow(cAllowSkip, cTitle, cCountdown, cTheme) != 0
}

func shouldForceOverlayShowFailForDebug() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("PAUSE_DEBUG_OVERLAY_FAIL")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func (darwinBreakOverlayController) Hide() {
	C.PauseBreakOverlayHide()
}

func (darwinBreakOverlayController) Destroy() {
	overlayCallbackMu.Lock()
	overlayCallback = nil
	overlayCallbackMu.Unlock()
	C.PauseBreakOverlayDestroy()
}

func (darwinBreakOverlayController) IsNative() bool {
	return true
}

//export overlaySkipCallbackGo
func overlaySkipCallbackGo() {
	overlayCallbackMu.RLock()
	cb := overlayCallback
	overlayCallbackMu.RUnlock()
	if cb != nil {
		cb()
	}
}
