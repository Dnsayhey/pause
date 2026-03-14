//go:build wails && windows

package desktop

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	msgOverlayApply = wmApp + 101

	ovlClassNamePrefix = "PauseOverlayWindowClass_"
	ovlButtonID        = 4001

	wsPopup      = 0x80000000
	wsChild      = 0x40000000
	wsVisible    = 0x10000000
	bsPushButton = 0x00000000

	wsExTopmost    = 0x00000008
	wsExToolWindow = 0x00000080

	swHide = 0
	swShow = 5

	hwndTopmost = ^uintptr(0) // (HWND)-1

	swpNoActivate = 0x0010
	swpShowWindow = 0x0040
	swpHideWindow = 0x0080
	swpNoZOrder   = 0x0004

	smXVirtualScreen  = 76
	smYVirtualScreen  = 77
	smCXVirtualScreen = 78
	smCYVirtualScreen = 79

	dtCenter     = 0x00000001
	dtVCenter    = 0x00000004
	dtSingleLine = 0x00000020
	dtNoPrefix   = 0x00000800

	colorBlack = 0x000000
	colorWhite = 0xFFFFFF
)

var (
	procShowWindow       = user32DLL.NewProc("ShowWindow")
	procSetWindowPos     = user32DLL.NewProc("SetWindowPos")
	procGetSystemMetrics = user32DLL.NewProc("GetSystemMetrics")
	procMoveWindow       = user32DLL.NewProc("MoveWindow")
	procInvalidateRect   = user32DLL.NewProc("InvalidateRect")
	procBeginPaint       = user32DLL.NewProc("BeginPaint")
	procEndPaint         = user32DLL.NewProc("EndPaint")
	procDrawTextW        = user32DLL.NewProc("DrawTextW")
	procSetTextColor     = user32DLL.NewProc("SetTextColor")
	procSetBkMode        = user32DLL.NewProc("SetBkMode")
	procCreateSolidBrush = user32DLL.NewProc("CreateSolidBrush")
	procFillRect         = user32DLL.NewProc("FillRect")
	procDeleteObject     = user32DLL.NewProc("DeleteObject")
	procSetWindowTextW   = user32DLL.NewProc("SetWindowTextW")
)

type windowsBreakOverlayController struct {
	mu sync.RWMutex

	onSkip func()

	allowSkip       bool
	skipButtonTitle string
	countdownText   string
	theme           string
	visible         bool

	hwnd       uintptr
	buttonHwnd uintptr
	started    bool
	startOnce  sync.Once
	ready      chan bool
	done       chan struct{}
}

type ovlWndClassEx struct {
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

type ovlRect struct {
	left   int32
	top    int32
	right  int32
	bottom int32
}

type ovlPaintStruct struct {
	hdc         uintptr
	fErase      int32
	rcPaint     ovlRect
	fRestore    int32
	fIncUpdate  int32
	rgbReserved [32]byte
}

var (
	overlayWndProcMu sync.RWMutex
	overlayByHwnd    = map[uintptr]*windowsBreakOverlayController{}
	overlayWndProc   = syscall.NewCallback(windowsOverlayWndProc)
)

func NewBreakOverlayController() BreakOverlayController {
	return &windowsBreakOverlayController{
		ready: make(chan bool, 1),
		done:  make(chan struct{}),
	}
}

func (c *windowsBreakOverlayController) Init(onSkip func()) {
	c.mu.Lock()
	c.onSkip = onSkip
	c.mu.Unlock()

	c.startOnce.Do(func() {
		go c.loop()
	})

	select {
	case ok := <-c.ready:
		c.mu.Lock()
		c.started = ok
		c.mu.Unlock()
	case <-time.After(2 * time.Second):
		c.mu.Lock()
		c.started = false
		c.mu.Unlock()
	}
}

func (c *windowsBreakOverlayController) Show(allowSkip bool, skipButtonTitle string, countdownText string, theme string) bool {
	c.mu.Lock()
	c.allowSkip = allowSkip
	c.skipButtonTitle = fallbackOverlay(skipButtonTitle, "Emergency Skip")
	c.countdownText = strings.TrimSpace(countdownText)
	c.theme = normalizeOverlayTheme(theme)
	c.visible = true
	hwnd := c.hwnd
	started := c.started
	c.mu.Unlock()

	if hwnd != 0 {
		postMessage(hwnd, msgOverlayApply, 0, 0)
	}
	return started
}

func (c *windowsBreakOverlayController) Hide() {
	c.mu.Lock()
	c.visible = false
	hwnd := c.hwnd
	c.mu.Unlock()
	if hwnd != 0 {
		postMessage(hwnd, msgOverlayApply, 0, 0)
	}
}

func (c *windowsBreakOverlayController) Destroy() {
	c.mu.RLock()
	hwnd := c.hwnd
	c.mu.RUnlock()
	if hwnd != 0 {
		postMessage(hwnd, wmClose, 0, 0)
		<-c.done
	}
}

func (c *windowsBreakOverlayController) IsNative() bool {
	return true
}

func (c *windowsBreakOverlayController) loop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(c.done)

	className := fmt.Sprintf("%s%d", ovlClassNamePrefix, uintptr(unsafe.Pointer(c)))
	classNamePtr, err := syscall.UTF16PtrFromString(className)
	if err != nil {
		c.ready <- false
		return
	}

	wc := ovlWndClassEx{
		cbSize:        uint32(unsafe.Sizeof(ovlWndClassEx{})),
		lpfnWndProc:   overlayWndProc,
		lpszClassName: classNamePtr,
	}
	if ret, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); ret == 0 {
		c.ready <- false
		return
	}

	x := getSystemMetric(smXVirtualScreen)
	y := getSystemMetric(smYVirtualScreen)
	w := getSystemMetric(smCXVirtualScreen)
	h := getSystemMetric(smCYVirtualScreen)
	if w <= 0 || h <= 0 {
		w = 1920
		h = 1080
	}

	windowNamePtr, _ := syscall.UTF16PtrFromString("PauseOverlay")
	hwnd, _, _ := procCreateWindowExW.Call(
		wsExTopmost|wsExToolWindow,
		uintptr(unsafe.Pointer(classNamePtr)),
		uintptr(unsafe.Pointer(windowNamePtr)),
		wsPopup,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		0,
		0,
		0,
		0,
	)
	if hwnd == 0 {
		c.ready <- false
		return
	}

	btnClass, _ := syscall.UTF16PtrFromString("BUTTON")
	btnText, _ := syscall.UTF16PtrFromString("Emergency Skip")
	btnHwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(btnClass)),
		uintptr(unsafe.Pointer(btnText)),
		wsChild|wsVisible|bsPushButton,
		uintptr(w/2-90),
		uintptr(h/2+24),
		180,
		40,
		hwnd,
		uintptr(ovlButtonID),
		0,
		0,
	)
	if btnHwnd == 0 {
		_, _, _ = procDestroyWindow.Call(hwnd)
		c.ready <- false
		return
	}

	c.mu.Lock()
	c.hwnd = hwnd
	c.buttonHwnd = btnHwnd
	c.started = true
	c.mu.Unlock()

	overlayWndProcMu.Lock()
	overlayByHwnd[hwnd] = c
	overlayWndProcMu.Unlock()

	c.ready <- true
	postMessage(hwnd, msgOverlayApply, 0, 0)

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

func (c *windowsBreakOverlayController) apply(hwnd uintptr) {
	c.mu.RLock()
	visible := c.visible
	allowSkip := c.allowSkip
	skipText := fallbackOverlay(c.skipButtonTitle, "Emergency Skip")
	btnHwnd := c.buttonHwnd
	c.mu.RUnlock()

	x := getSystemMetric(smXVirtualScreen)
	y := getSystemMetric(smYVirtualScreen)
	w := getSystemMetric(smCXVirtualScreen)
	h := getSystemMetric(smCYVirtualScreen)
	if w <= 0 || h <= 0 {
		w = 1920
		h = 1080
	}

	_, _, _ = procSetWindowPos.Call(
		hwnd,
		hwndTopmost,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		swpNoActivate,
	)

	if btnHwnd != 0 {
		btnW := 180
		btnH := 40
		btnX := (w - btnW) / 2
		btnY := h/2 + 24
		_, _, _ = procMoveWindow.Call(btnHwnd, uintptr(btnX), uintptr(btnY), uintptr(btnW), uintptr(btnH), 1)

		txt, _ := syscall.UTF16PtrFromString(skipText)
		_, _, _ = procSetWindowTextW.Call(btnHwnd, uintptr(unsafe.Pointer(txt)))
		if allowSkip {
			_, _, _ = procShowWindow.Call(btnHwnd, swShow)
		} else {
			_, _, _ = procShowWindow.Call(btnHwnd, swHide)
		}
	}

	if visible {
		_, _, _ = procSetWindowPos.Call(
			hwnd,
			hwndTopmost,
			uintptr(x), uintptr(y), uintptr(w), uintptr(h),
			swpNoActivate|swpShowWindow,
		)
		_, _, _ = procShowWindow.Call(hwnd, swShow)
	} else {
		_, _, _ = procSetWindowPos.Call(
			hwnd,
			hwndTopmost,
			uintptr(x), uintptr(y), uintptr(w), uintptr(h),
			swpNoActivate|swpHideWindow|swpNoZOrder,
		)
		_, _, _ = procShowWindow.Call(hwnd, swHide)
	}
	_, _, _ = procInvalidateRect.Call(hwnd, 0, 1)
}

func (c *windowsBreakOverlayController) handleSkipClick() {
	c.mu.RLock()
	if !c.allowSkip {
		c.mu.RUnlock()
		return
	}
	cb := c.onSkip
	c.mu.RUnlock()

	if cb != nil {
		go cb()
	}
}

func (c *windowsBreakOverlayController) paint(hwnd uintptr) {
	c.mu.RLock()
	text := c.countdownText
	theme := c.theme
	c.mu.RUnlock()
	if strings.TrimSpace(text) == "" {
		text = ""
	}

	bg := colorBlack
	fg := colorWhite
	if theme == "light" {
		bg = colorWhite
		fg = colorBlack
	}

	var ps ovlPaintStruct
	hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}

	brush, _, _ := procCreateSolidBrush.Call(uintptr(bg))
	if brush != 0 {
		_, _, _ = procFillRect.Call(hdc, uintptr(unsafe.Pointer(&ps.rcPaint)), brush)
		_, _, _ = procDeleteObject.Call(brush)
	}

	// TRANSPARENT background for text drawing.
	_, _, _ = procSetBkMode.Call(hdc, 1)
	_, _, _ = procSetTextColor.Call(hdc, uintptr(fg))

	textRect := ovlRect{
		left:   ps.rcPaint.left,
		top:    ps.rcPaint.top,
		right:  ps.rcPaint.right,
		bottom: ps.rcPaint.bottom - 60,
	}
	utf16Text := syscall.StringToUTF16(text)
	if len(utf16Text) == 0 {
		utf16Text = []uint16{0}
	}
	_, _, _ = procDrawTextW.Call(
		hdc,
		uintptr(unsafe.Pointer(&utf16Text[0])),
		^uintptr(0),
		uintptr(unsafe.Pointer(&textRect)),
		dtCenter|dtVCenter|dtSingleLine|dtNoPrefix,
	)

	_, _, _ = procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
}

func (c *windowsBreakOverlayController) destroy(hwnd uintptr) {
	overlayWndProcMu.Lock()
	delete(overlayByHwnd, hwnd)
	overlayWndProcMu.Unlock()

	c.mu.Lock()
	c.hwnd = 0
	c.buttonHwnd = 0
	c.mu.Unlock()
	_, _, _ = procPostQuitMessage.Call(0)
}

func windowsOverlayWndProc(hwnd uintptr, msgID uint32, wParam uintptr, lParam uintptr) uintptr {
	overlayWndProcMu.RLock()
	ctrl := overlayByHwnd[hwnd]
	overlayWndProcMu.RUnlock()
	if ctrl == nil {
		ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msgID), wParam, lParam)
		return ret
	}

	switch msgID {
	case msgOverlayApply:
		ctrl.apply(hwnd)
		return 0
	case 0x0111: // WM_COMMAND
		if loWord(wParam) == ovlButtonID {
			ctrl.handleSkipClick()
			return 0
		}
	case 0x000F: // WM_PAINT
		ctrl.paint(hwnd)
		return 0
	case wmClose:
		_, _, _ = procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		ctrl.destroy(hwnd)
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msgID), wParam, lParam)
	return ret
}

func getSystemMetric(index int) int {
	v, _, _ := procGetSystemMetrics.Call(uintptr(index))
	return int(int32(v))
}

func loWord(v uintptr) int {
	return int(uint16(v & 0xFFFF))
}

func normalizeOverlayTheme(theme string) string {
	theme = strings.TrimSpace(strings.ToLower(theme))
	if theme != "light" {
		return "dark"
	}
	return "light"
}

func fallbackOverlay(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
