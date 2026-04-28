//go:build darwin && wails

package desktop

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Foundation

#include <stdlib.h>

void PauseBreakOverlayInit(void);
	int PauseBreakOverlayShow(int allowSkip, const char *skipButtonTitle, int allowPostpone, const char *postponeButtonTitle, const char *countdownText, const char *messageText, const char *theme);
void PauseBreakOverlayHide(void);
void PauseBreakOverlayDestroy(void);
*/
import "C"

import (
	"os"
	"strings"
	"sync"
	"unsafe"

	_ "pause/internal/desktop/macbridge"
)

type darwinBreakOverlayController struct{}

var (
	overlayCallbackMu       sync.RWMutex
	overlayCallback         func()
	overlayPostponeCallback func()
)

func NewBreakOverlayController() BreakOverlayController {
	return darwinBreakOverlayController{}
}

func (darwinBreakOverlayController) Init(onSkip func(), onPostpone func()) {
	overlayCallbackMu.Lock()
	overlayCallback = onSkip
	overlayPostponeCallback = onPostpone
	overlayCallbackMu.Unlock()
	C.PauseBreakOverlayInit()
}

func (darwinBreakOverlayController) Show(allowSkip bool, skipButtonTitle string, allowPostpone bool, postponeButtonTitle string, countdownText string, messageText string, theme string) bool {
	if shouldForceOverlayShowFailForDebug() {
		return false
	}

	cTitle := C.CString(skipButtonTitle)
	cPostponeTitle := C.CString(postponeButtonTitle)
	cCountdown := C.CString(countdownText)
	cMessage := C.CString(messageText)
	cTheme := C.CString(theme)
	defer C.free(unsafe.Pointer(cTitle))
	defer C.free(unsafe.Pointer(cPostponeTitle))
	defer C.free(unsafe.Pointer(cCountdown))
	defer C.free(unsafe.Pointer(cMessage))
	defer C.free(unsafe.Pointer(cTheme))

	cAllowSkip := C.int(0)
	if allowSkip {
		cAllowSkip = 1
	}
	cAllowPostpone := C.int(0)
	if allowPostpone {
		cAllowPostpone = 1
	}
	return C.PauseBreakOverlayShow(cAllowSkip, cTitle, cAllowPostpone, cPostponeTitle, cCountdown, cMessage, cTheme) != 0
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
	overlayPostponeCallback = nil
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

//export overlayPostponeCallbackGo
func overlayPostponeCallbackGo() {
	overlayCallbackMu.RLock()
	cb := overlayPostponeCallback
	overlayCallbackMu.RUnlock()
	if cb != nil {
		cb()
	}
}
