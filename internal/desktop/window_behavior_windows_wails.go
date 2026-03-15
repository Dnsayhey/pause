//go:build windows && wails

package desktop

import (
	"context"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	gwlStyleIndex = ^uintptr(15) // -16 casted to uintptr for WinAPI
	wsMinimizeBox = 0x00020000
	wsMaximizeBox = 0x00010000

	wbSwpNoSize       = 0x0001
	wbSwpNoMove       = 0x0002
	wbSwpNoZOrder     = 0x0004
	wbSwpNoActivate   = 0x0010
	wbSwpFrameChanged = 0x0020
)

var (
	windowBehaviorUser32 = syscall.NewLazyDLL("user32.dll")

	procFindWindowW      = windowBehaviorUser32.NewProc("FindWindowW")
	procGetWindowLongPtrW = windowBehaviorUser32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW = windowBehaviorUser32.NewProc("SetWindowLongPtrW")
	procSetWindowPosWB    = windowBehaviorUser32.NewProc("SetWindowPos")
)

func configureDesktopWindowBehavior() {
	title, _ := syscall.UTF16PtrFromString("Pause")
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(title)))
	if hwnd == 0 {
		return
	}

	style, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlStyleIndex)
	if style == 0 {
		return
	}
	newStyle := style &^ uintptr(wsMinimizeBox|wsMaximizeBox)
	if newStyle == style {
		return
	}
	_, _, _ = procSetWindowLongPtrW.Call(hwnd, gwlStyleIndex, newStyle)
	_, _, _ = procSetWindowPosWB.Call(
		hwnd,
		0,
		0, 0, 0, 0,
		wbSwpNoMove|wbSwpNoSize|wbSwpNoZOrder|wbSwpNoActivate|wbSwpFrameChanged,
	)
}

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
