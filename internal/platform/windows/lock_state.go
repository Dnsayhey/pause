//go:build windows

package windows

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	xwindows "golang.org/x/sys/windows"
)

const (
	wmWTSSessionChange       = 0x02B1
	wmDestroyLockStateWindow = 0x0002
	wtsNotifyForThisSession  = 0

	uoiName              = 2
	desktopReadObjects   = 0x0001
	desktopSwitchDesktop = 0x0100
)

var (
	wtsapi32DLL = syscall.NewLazyDLL("wtsapi32.dll")

	procWTSRegisterSessionNotification   = wtsapi32DLL.NewProc("WTSRegisterSessionNotification")
	procWTSUnRegisterSessionNotification = wtsapi32DLL.NewProc("WTSUnRegisterSessionNotification")

	procRegisterClassExWLockState      = user32DLL.NewProc("RegisterClassExW")
	procDefWindowProcWLockState        = user32DLL.NewProc("DefWindowProcW")
	procGetMessageWLockState           = user32DLL.NewProc("GetMessageW")
	procTranslateMessageLockState      = user32DLL.NewProc("TranslateMessage")
	procDispatchMessageWLockState      = user32DLL.NewProc("DispatchMessageW")
	procPostQuitMessageLockState       = user32DLL.NewProc("PostQuitMessage")
	procOpenInputDesktop               = user32DLL.NewProc("OpenInputDesktop")
	procCloseDesktop                   = user32DLL.NewProc("CloseDesktop")
	procGetUserObjectInformationW      = user32DLL.NewProc("GetUserObjectInformationW")
	windowsLockStateWndProc            = syscall.NewCallback(windowsLockStateWindowProc)
	windowsLockStateWindowRegistryMu   sync.RWMutex
	windowsLockStateProviderByWindowID = map[uintptr]*windowsLockSessionState{}
)

type windowsLockStateProvider struct {
	state *windowsLockSessionState
}

type windowsLockSessionState struct {
	startOnce sync.Once
	locked    atomic.Bool
}

type lockStateWindowClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type lockStateWindowMsg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      lockStateWindowPoint
}

type lockStateWindowPoint struct {
	x int32
	y int32
}

func newWindowsLockStateProvider() windowsLockStateProvider {
	state := &windowsLockSessionState{}
	state.locked.Store(probeWindowsSessionLocked())
	return windowsLockStateProvider{state: state}
}

func (p windowsLockStateProvider) IsScreenLocked() bool {
	if p.state == nil {
		return false
	}
	p.state.start()
	return p.state.locked.Load()
}

func (s *windowsLockSessionState) start() {
	s.startOnce.Do(func() {
		go s.loop()
	})
}

func (s *windowsLockSessionState) loop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	className := fmt.Sprintf("PauseLockStateWindowClass_%d", uintptr(unsafe.Pointer(s)))
	classNamePtr, err := syscall.UTF16PtrFromString(className)
	if err != nil {
		return
	}

	windowClass := lockStateWindowClassEx{
		cbSize:        uint32(unsafe.Sizeof(lockStateWindowClassEx{})),
		lpfnWndProc:   windowsLockStateWndProc,
		lpszClassName: classNamePtr,
	}
	if ret, _, _ := procRegisterClassExWLockState.Call(uintptr(unsafe.Pointer(&windowClass))); ret == 0 {
		return
	}

	windowNamePtr, _ := syscall.UTF16PtrFromString("PauseLockState")
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(classNamePtr)),
		uintptr(unsafe.Pointer(windowNamePtr)),
		0,
		0, 0, 0, 0,
		0,
		0,
		0,
		0,
	)
	if hwnd == 0 {
		return
	}

	windowsLockStateWindowRegistryMu.Lock()
	windowsLockStateProviderByWindowID[hwnd] = s
	windowsLockStateWindowRegistryMu.Unlock()

	if ret, _, _ := procWTSRegisterSessionNotification.Call(hwnd, wtsNotifyForThisSession); ret == 0 {
		windowsLockStateWindowRegistryMu.Lock()
		delete(windowsLockStateProviderByWindowID, hwnd)
		windowsLockStateWindowRegistryMu.Unlock()
		_, _, _ = procDestroyWindow.Call(hwnd)
		return
	}

	var message lockStateWindowMsg
	for {
		ret, _, _ := procGetMessageWLockState.Call(
			uintptr(unsafe.Pointer(&message)),
			0,
			0,
			0,
		)
		if int32(ret) <= 0 {
			break
		}
		_, _, _ = procTranslateMessageLockState.Call(uintptr(unsafe.Pointer(&message)))
		_, _, _ = procDispatchMessageWLockState.Call(uintptr(unsafe.Pointer(&message)))
	}
}

func windowsLockStateWindowProc(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) uintptr {
	windowsLockStateWindowRegistryMu.RLock()
	state := windowsLockStateProviderByWindowID[hwnd]
	windowsLockStateWindowRegistryMu.RUnlock()
	if state == nil {
		ret, _, _ := procDefWindowProcWLockState.Call(hwnd, uintptr(msg), wParam, lParam)
		return ret
	}

	switch msg {
	case wmWTSSessionChange:
		switch uint32(wParam) {
		case xwindows.WTS_SESSION_LOCK:
			state.locked.Store(true)
		case xwindows.WTS_SESSION_UNLOCK:
			state.locked.Store(false)
		}
		return 0
	case wmDestroyLockStateWindow:
		_, _, _ = procWTSUnRegisterSessionNotification.Call(hwnd)
		windowsLockStateWindowRegistryMu.Lock()
		delete(windowsLockStateProviderByWindowID, hwnd)
		windowsLockStateWindowRegistryMu.Unlock()
		_, _, _ = procPostQuitMessageLockState.Call(0)
		return 0
	default:
		ret, _, _ := procDefWindowProcWLockState.Call(hwnd, uintptr(msg), wParam, lParam)
		return ret
	}
}

func probeWindowsSessionLocked() bool {
	name, ok := currentInputDesktopName()
	if !ok {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "winlogon", "screen-saver":
		return true
	default:
		return false
	}
}

func currentInputDesktopName() (string, bool) {
	desktop, _, _ := procOpenInputDesktop.Call(
		0,
		0,
		desktopReadObjects|desktopSwitchDesktop,
	)
	if desktop == 0 {
		return "", false
	}
	defer func() {
		_, _, _ = procCloseDesktop.Call(desktop)
	}()

	var needed uint32
	_, _, _ = procGetUserObjectInformationW.Call(
		desktop,
		uoiName,
		0,
		0,
		uintptr(unsafe.Pointer(&needed)),
	)
	if needed == 0 {
		return "", false
	}

	buffer := make([]uint16, int((needed+1)/2))
	ret, _, _ := procGetUserObjectInformationW.Call(
		desktop,
		uoiName,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(needed),
		uintptr(unsafe.Pointer(&needed)),
	)
	if ret == 0 {
		return "", false
	}

	return syscall.UTF16ToString(buffer), true
}
