//go:build wails && windows

package desktop

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const (
	wmClose   = 0x0010
	wmDestroy = 0x0002
	wmUser    = 0x0400
	wmApp     = 0x8000

	msgTrayCallback = wmApp + 1
	msgTrayUpdate   = wmApp + 2

	wmLButtonUp  = 0x0202
	wmRButtonUp  = 0x0205
	wmContext    = 0x007B
	wmLButtonDbl = 0x0203

	nimAdd    = 0x00000000
	nimModify = 0x00000001
	nimDelete = 0x00000002

	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004

	idiApplication = 32512

	mfString    = 0x00000000
	mfSeparator = 0x00000800
	mfGray      = 0x00000001
	mfDisabled  = 0x00000002

	tpmLeftAlign   = 0x0000
	tpmBottomAlign = 0x0020
	tpmRightButton = 0x0002
	tpmReturnCmd   = 0x0100
)

const (
	menuBreakNow = 1001
	menuPause    = 1002
	menuPause30  = 1003
	menuResume   = 1004
	menuOpen     = 1005
	menuAbout    = 1006
	menuQuit     = 1007
)

var (
	user32DLL = syscall.NewLazyDLL("user32.dll")
	shell32   = syscall.NewLazyDLL("shell32.dll")

	procRegisterClassExW = user32DLL.NewProc("RegisterClassExW")
	procCreateWindowExW  = user32DLL.NewProc("CreateWindowExW")
	procDefWindowProcW   = user32DLL.NewProc("DefWindowProcW")
	procDestroyWindow    = user32DLL.NewProc("DestroyWindow")
	procGetMessageW      = user32DLL.NewProc("GetMessageW")
	procTranslateMessage = user32DLL.NewProc("TranslateMessage")
	procDispatchMessageW = user32DLL.NewProc("DispatchMessageW")
	procPostMessageW     = user32DLL.NewProc("PostMessageW")
	procPostQuitMessage  = user32DLL.NewProc("PostQuitMessage")
	procLoadIconW        = user32DLL.NewProc("LoadIconW")
	procCreatePopupMenu  = user32DLL.NewProc("CreatePopupMenu")
	procAppendMenuW      = user32DLL.NewProc("AppendMenuW")
	procTrackPopupMenu   = user32DLL.NewProc("TrackPopupMenu")
	procDestroyMenu      = user32DLL.NewProc("DestroyMenu")
	procGetCursorPos     = user32DLL.NewProc("GetCursorPos")
	procSetForegroundWnd = user32DLL.NewProc("SetForegroundWindow")

	procShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")
)

type windowsStatusBarController struct {
	mu sync.RWMutex

	onAction func(int)
	locale   StatusBarLocaleStrings

	status    string
	countdown string
	title     string
	paused    bool

	hwnd      uintptr
	startOnce sync.Once
	done      chan struct{}
}

type wndClassEx struct {
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

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type point struct {
	x int32
	y int32
}

type notifyIconData struct {
	cbSize            uint32
	hWnd              uintptr
	uID               uint32
	uFlags            uint32
	uCallbackMessage  uint32
	hIcon             uintptr
	szTip             [128]uint16
	dwState           uint32
	dwStateMask       uint32
	szInfo            [256]uint16
	uTimeoutOrVersion uint32
	szInfoTitle       [64]uint16
	dwInfoFlags       uint32
}

var (
	windowsTrayWndProcMu sync.RWMutex
	windowsTrayByHwnd    = map[uintptr]*windowsStatusBarController{}
	windowsTrayWndProc   = syscall.NewCallback(windowsStatusbarWndProc)
)

func NewStatusBarController() StatusBarController {
	return &windowsStatusBarController{
		done: make(chan struct{}),
	}
}

func (c *windowsStatusBarController) Init(onAction func(actionID int)) {
	c.mu.Lock()
	c.onAction = onAction
	c.mu.Unlock()

	c.startOnce.Do(func() {
		go c.loop()
	})
}

func (c *windowsStatusBarController) Update(status, countdown, title string, paused bool, _ float64) {
	c.mu.Lock()
	c.status = strings.TrimSpace(status)
	c.countdown = strings.TrimSpace(countdown)
	c.title = strings.TrimSpace(title)
	c.paused = paused
	hwnd := c.hwnd
	c.mu.Unlock()

	if hwnd != 0 {
		postMessage(hwnd, msgTrayUpdate, 0, 0)
	}
}

func (c *windowsStatusBarController) SetLocale(strings StatusBarLocaleStrings) {
	c.mu.Lock()
	c.locale = strings
	c.mu.Unlock()
}

func (c *windowsStatusBarController) Destroy() {
	c.mu.RLock()
	hwnd := c.hwnd
	c.mu.RUnlock()
	if hwnd != 0 {
		postMessage(hwnd, wmClose, 0, 0)
		<-c.done
	}
}

func (c *windowsStatusBarController) loop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(c.done)

	className := fmt.Sprintf("PauseTrayWindowClass_%d", uintptr(unsafe.Pointer(c)))
	classNamePtr, err := syscall.UTF16PtrFromString(className)
	if err != nil {
		return
	}

	wc := wndClassEx{
		cbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		lpfnWndProc:   windowsTrayWndProc,
		lpszClassName: classNamePtr,
	}
	if ret, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); ret == 0 {
		return
	}

	windowNamePtr, _ := syscall.UTF16PtrFromString("PauseTray")
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

	c.mu.Lock()
	c.hwnd = hwnd
	c.mu.Unlock()

	windowsTrayWndProcMu.Lock()
	windowsTrayByHwnd[hwnd] = c
	windowsTrayWndProcMu.Unlock()

	if !c.addTrayIcon(hwnd) {
		_, _, _ = procDestroyWindow.Call(hwnd)
		return
	}

	var m msg
	for {
		ret, _, _ := procGetMessageW.Call(
			uintptr(unsafe.Pointer(&m)),
			0,
			0,
			0,
		)
		if int32(ret) <= 0 {
			break
		}
		_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func (c *windowsStatusBarController) addTrayIcon(hwnd uintptr) bool {
	icon, _, _ := procLoadIconW.Call(0, uintptr(idiApplication))
	nid := notifyIconData{
		cbSize:           uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:             hwnd,
		uID:              1,
		uFlags:           nifMessage | nifIcon | nifTip,
		uCallbackMessage: msgTrayCallback,
		hIcon:            icon,
	}
	copyUTF16(nid.szTip[:], "Pause")
	return shellNotifyIcon(nimAdd, &nid)
}

func (c *windowsStatusBarController) updateTooltip(hwnd uintptr) {
	c.mu.RLock()
	tip := composeTrayTip(c.status, c.countdown, c.title)
	c.mu.RUnlock()

	nid := notifyIconData{
		cbSize: uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:   hwnd,
		uID:    1,
		uFlags: nifTip,
	}
	copyUTF16(nid.szTip[:], tip)
	_ = shellNotifyIcon(nimModify, &nid)
}

func composeTrayTip(status, countdown, title string) string {
	parts := make([]string, 0, 3)
	if status != "" {
		parts = append(parts, status)
	}
	if countdown != "" {
		parts = append(parts, countdown)
	}
	if title != "" {
		parts = append(parts, title)
	}
	if len(parts) == 0 {
		return "Pause"
	}
	return strings.Join(parts, " | ")
}

func (c *windowsStatusBarController) showContextMenu(hwnd uintptr) {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer func() {
		_, _, _ = procDestroyMenu.Call(menu)
	}()

	c.mu.RLock()
	paused := c.paused
	locale := c.locale
	c.mu.RUnlock()

	labelBreakNow := fallback(locale.BreakNowButton, "Break now")
	labelPause := fallback(locale.PauseButton, "Pause")
	labelPause30 := fallback(locale.Pause30Button, "Pause 30m")
	labelResume := fallback(locale.ResumeButton, "Resume")
	labelOpen := fallback(locale.OpenAppButton, "Open")
	labelAbout := fallback(locale.AboutMenuItem, "About")
	labelQuit := fallback(locale.QuitMenuItem, "Quit")

	addMenuText(menu, menuBreakNow, labelBreakNow, false)
	addMenuText(menu, menuPause, labelPause, paused)
	addMenuText(menu, menuPause30, labelPause30, paused)
	addMenuText(menu, menuResume, labelResume, !paused)
	addMenuSeparator(menu)
	addMenuText(menu, menuOpen, labelOpen, false)
	addMenuText(menu, menuAbout, labelAbout, false)
	addMenuSeparator(menu)
	addMenuText(menu, menuQuit, labelQuit, false)

	var p point
	_, _, _ = procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	_, _, _ = procSetForegroundWnd.Call(hwnd)

	cmd, _, _ := procTrackPopupMenu.Call(
		menu,
		uintptr(tpmLeftAlign|tpmBottomAlign|tpmRightButton|tpmReturnCmd),
		uintptr(p.x),
		uintptr(p.y),
		0,
		hwnd,
		0,
	)
	c.dispatchMenuCommand(int(cmd))

	// Required by Windows so the menu closes consistently when clicking elsewhere.
	postMessage(hwnd, wmUser, 0, 0)
}

func addMenuText(menu uintptr, id int, label string, disabled bool) {
	flags := uintptr(mfString)
	if disabled {
		flags |= mfDisabled | mfGray
	}
	ptr, err := syscall.UTF16PtrFromString(label)
	if err != nil {
		return
	}
	_, _, _ = procAppendMenuW.Call(
		menu,
		flags,
		uintptr(id),
		uintptr(unsafe.Pointer(ptr)),
	)
}

func addMenuSeparator(menu uintptr) {
	_, _, _ = procAppendMenuW.Call(menu, mfSeparator, 0, 0)
}

func fallback(value, fallbackText string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallbackText
	}
	return value
}

func (c *windowsStatusBarController) dispatchMenuCommand(id int) {
	var action int
	switch id {
	case menuBreakNow:
		action = StatusBarActionBreakNow
	case menuPause:
		action = StatusBarActionPause
	case menuPause30:
		action = StatusBarActionPause30
	case menuResume:
		action = StatusBarActionResume
	case menuOpen, menuAbout:
		action = StatusBarActionOpenWindow
	case menuQuit:
		action = StatusBarActionQuit
	default:
		return
	}

	c.mu.RLock()
	cb := c.onAction
	c.mu.RUnlock()
	if cb != nil {
		cb(action)
	}
}

func (c *windowsStatusBarController) destroy(hwnd uintptr) {
	nid := notifyIconData{
		cbSize: uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:   hwnd,
		uID:    1,
	}
	_ = shellNotifyIcon(nimDelete, &nid)

	windowsTrayWndProcMu.Lock()
	delete(windowsTrayByHwnd, hwnd)
	windowsTrayWndProcMu.Unlock()

	c.mu.Lock()
	c.hwnd = 0
	c.mu.Unlock()
	_, _, _ = procPostQuitMessage.Call(0)
}

func windowsStatusbarWndProc(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) uintptr {
	windowsTrayWndProcMu.RLock()
	ctrl := windowsTrayByHwnd[hwnd]
	windowsTrayWndProcMu.RUnlock()
	if ctrl == nil {
		ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return ret
	}

	switch msg {
	case msgTrayUpdate:
		ctrl.updateTooltip(hwnd)
		return 0
	case msgTrayCallback:
		switch uint32(lParam) {
		case wmLButtonUp, wmLButtonDbl:
			ctrl.dispatchMenuCommand(menuOpen)
		case wmRButtonUp, wmContext:
			ctrl.showContextMenu(hwnd)
		}
		return 0
	case wmClose:
		_, _, _ = procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		ctrl.destroy(hwnd)
		return 0
	default:
		ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return ret
	}
}

func shellNotifyIcon(message uint32, data *notifyIconData) bool {
	ret, _, _ := procShellNotifyIconW.Call(
		uintptr(message),
		uintptr(unsafe.Pointer(data)),
	)
	return ret != 0
}

func copyUTF16(dst []uint16, value string) {
	if len(dst) == 0 {
		return
	}
	encoded := syscall.StringToUTF16(value)
	if len(encoded) > len(dst) {
		encoded = encoded[:len(dst)]
		encoded[len(dst)-1] = 0
	}
	copy(dst, encoded)
}

func postMessage(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) {
	_, _, _ = procPostMessageW.Call(hwnd, uintptr(msg), wParam, lParam)
}
