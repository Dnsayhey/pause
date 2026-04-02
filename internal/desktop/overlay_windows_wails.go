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
	wsExLayered    = 0x00080000

	swHide = 0
	swShow = 5

	hwndTopmost = ^uintptr(0) // (HWND)-1

	swpNoActivate = 0x0010
	swpShowWindow = 0x0040
	swpHideWindow = 0x0080
	swpNoZOrder   = 0x0004
	gwlExStyleIdx = ^uintptr(19) // -20 casted to uintptr for WinAPI

	smXVirtualScreen        = 76
	smYVirtualScreen        = 77
	smCXVirtualScreen       = 78
	smCYVirtualScreen       = 79
	monitorDefaultToNearest = 0x00000002
	smCXScreen              = 0
	smCYScreen              = 1

	dtCenter     = 0x00000001
	dtVCenter    = 0x00000004
	dtSingleLine = 0x00000020
	dtNoPrefix   = 0x00000800
	dtWordBreak  = 0x00000010

	colorBlack = 0x000000
	colorWhite = 0xFFFFFF
	// Keep Windows overlay button palette aligned with macOS: softer neutrals,
	// no pure black/white blocks that feel harsh on full-screen overlays.
	colorButtonDarkBg         = 0x212428
	colorButtonDarkBgHover    = 0x2B2E36
	colorButtonDarkBgPressed  = 0x171A1F
	colorButtonDarkFg         = 0xEBEDF2
	colorButtonLightBg        = 0xF5F5F5
	colorButtonLightBgHover   = 0xEBEBEB
	colorButtonLightBgPressed = 0xD6D9E0
	colorButtonLightFg        = 0x141414
	alphaButtonBase           = 220
	alphaButtonHover          = 236
	alphaButtonPressed        = 255

	fwBold                   = 700
	wmLButtonDownLocal       = 0x0201
	wmLButtonUpLocal         = 0x0202
	wmMouseMoveLocal         = 0x0200
	wmMouseLeaveLocal        = 0x02A3
	wmKeyDownLocal           = 0x0100
	wmSysKeyDown             = 0x0104
	tmeLeave                 = 0x00000002
	overlayFadeInDurationMs  = 1000
	overlayFadeOutDurationMs = 1000
	overlayFadeStepMs        = 16
	lwaAlpha                 = 0x00000002
	idcArrow                 = 32512
	vkEscape                 = 0x1B
	vkF4                     = 0x73
	vkSpace                  = 0x20
	vkW                      = 0x57
	vkControl                = 0x11
	vkMenu                   = 0x12 // Alt
)

var (
	gdi32DLL = syscall.NewLazyDLL("gdi32.dll")

	procShowWindow                 = user32DLL.NewProc("ShowWindow")
	procLoadCursorW                = user32DLL.NewProc("LoadCursorW")
	procGetKeyState                = user32DLL.NewProc("GetKeyState")
	procSetCapture                 = user32DLL.NewProc("SetCapture")
	procReleaseCapture             = user32DLL.NewProc("ReleaseCapture")
	procTrackMouseEvent            = user32DLL.NewProc("TrackMouseEvent")
	procGetWindowLongPtrWOvl       = user32DLL.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrWOvl       = user32DLL.NewProc("SetWindowLongPtrW")
	procSetLayeredWindowAttributes = user32DLL.NewProc("SetLayeredWindowAttributes")
	procSetWindowPos               = user32DLL.NewProc("SetWindowPos")
	procSetActiveWindow            = user32DLL.NewProc("SetActiveWindow")
	procSetFocus                   = user32DLL.NewProc("SetFocus")
	procGetSystemMetrics           = user32DLL.NewProc("GetSystemMetrics")
	procMoveWindow                 = user32DLL.NewProc("MoveWindow")
	procGetClientRect              = user32DLL.NewProc("GetClientRect")
	procUpdateWindow               = user32DLL.NewProc("UpdateWindow")
	procEnumDisplayMonitors        = user32DLL.NewProc("EnumDisplayMonitors")
	procMonitorFromWindow          = user32DLL.NewProc("MonitorFromWindow")
	procGetMonitorInfoW            = user32DLL.NewProc("GetMonitorInfoW")
	procInvalidateRect             = user32DLL.NewProc("InvalidateRect")
	procBeginPaint                 = user32DLL.NewProc("BeginPaint")
	procEndPaint                   = user32DLL.NewProc("EndPaint")
	procDrawTextW                  = user32DLL.NewProc("DrawTextW")
	procSetTextColor               = gdi32DLL.NewProc("SetTextColor")
	procSetBkMode                  = gdi32DLL.NewProc("SetBkMode")
	procCreateFontW                = gdi32DLL.NewProc("CreateFontW")
	procSelectObject               = gdi32DLL.NewProc("SelectObject")
	procCreateSolidBrush           = gdi32DLL.NewProc("CreateSolidBrush")
	procFillRect                   = user32DLL.NewProc("FillRect")
	procDeleteObject               = gdi32DLL.NewProc("DeleteObject")
	procSetWindowTextW             = user32DLL.NewProc("SetWindowTextW")
)

type windowsBreakOverlayController struct {
	mu sync.RWMutex

	onSkip func()

	allowSkip             bool
	skipButtonTitle       string
	countdownText         string
	theme                 string
	visible               bool
	className             string
	focusOwnerHwnd        uintptr
	activationPending     bool
	emergencySkipVisible  bool
	lastBlockedShortcutAt time.Time
	blockedShortcutCount  int

	windows   []windowsOverlayWindow
	started   bool
	startOnce sync.Once
	ready     chan bool
	done      chan struct{}
}

type windowsOverlayWindow struct {
	hwnd          uintptr
	x             int
	y             int
	w             int
	h             int
	primary       bool
	buttonRect    ovlRect
	buttonHot     bool
	buttonPressed bool
	trackingMouse bool
	shown         bool
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

type ovlMonitorInfo struct {
	cbSize    uint32
	rcMonitor ovlRect
	rcWork    ovlRect
	dwFlags   uint32
}

type ovlMonitorBounds struct {
	x       int
	y       int
	w       int
	h       int
	primary bool
}

type ovlTrackMouseEvent struct {
	cbSize      uint32
	dwFlags     uint32
	hwndTrack   uintptr
	dwHoverTime uint32
}

var (
	overlayWndProcMu     sync.RWMutex
	overlayByHwnd        = map[uintptr]*windowsBreakOverlayController{}
	overlayWndProc       = syscall.NewCallback(windowsOverlayWndProc)
	overlayMonitorEnumMu sync.Mutex
	overlayMonitorsOut   *[]ovlMonitorBounds
	overlayMonitorEnumCb = syscall.NewCallback(windowsMonitorEnumProc)
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
	wasVisible := c.visible
	c.allowSkip = allowSkip
	c.skipButtonTitle = fallbackOverlay(skipButtonTitle, "Emergency Skip")
	c.countdownText = strings.TrimSpace(countdownText)
	c.theme = normalizeOverlayTheme(theme)
	c.visible = true
	if !wasVisible {
		c.focusOwnerHwnd = c.chooseFocusOwnerWindowLocked()
		c.activationPending = true
		c.emergencySkipVisible = false
		c.lastBlockedShortcutAt = time.Time{}
		c.blockedShortcutCount = 0
	}
	windows := append([]windowsOverlayWindow(nil), c.windows...)
	started := c.started
	c.mu.Unlock()

	for _, wnd := range windows {
		if wnd.hwnd != 0 {
			postMessage(wnd.hwnd, msgOverlayApply, 0, 0)
		}
	}
	return started
}

func (c *windowsBreakOverlayController) Hide() {
	c.mu.Lock()
	c.visible = false
	c.focusOwnerHwnd = 0
	c.activationPending = false
	c.emergencySkipVisible = false
	c.lastBlockedShortcutAt = time.Time{}
	c.blockedShortcutCount = 0
	windows := append([]windowsOverlayWindow(nil), c.windows...)
	c.mu.Unlock()
	for _, wnd := range windows {
		if wnd.hwnd != 0 {
			postMessage(wnd.hwnd, msgOverlayApply, 0, 0)
		}
	}
}

func (c *windowsBreakOverlayController) Destroy() {
	c.mu.RLock()
	windows := append([]windowsOverlayWindow(nil), c.windows...)
	c.mu.RUnlock()
	if len(windows) > 0 {
		for _, wnd := range windows {
			if wnd.hwnd != 0 {
				postMessage(wnd.hwnd, wmClose, 0, 0)
			}
		}
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
		hCursor:       loadArrowCursor(),
		lpszClassName: classNamePtr,
	}
	if ret, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); ret == 0 {
		c.ready <- false
		return
	}
	c.mu.Lock()
	c.className = className
	c.mu.Unlock()

	monitors := listOverlayMonitors()
	if len(monitors) == 0 {
		c.ready <- false
		return
	}

	created := make([]windowsOverlayWindow, 0, len(monitors))
	windowNamePtr, _ := syscall.UTF16PtrFromString("PauseOverlay")

	for _, m := range monitors {
		hwnd, _, _ := procCreateWindowExW.Call(
			wsExTopmost|wsExToolWindow,
			uintptr(unsafe.Pointer(classNamePtr)),
			uintptr(unsafe.Pointer(windowNamePtr)),
			wsPopup,
			uintptr(m.x), uintptr(m.y), uintptr(m.w), uintptr(m.h),
			0,
			0,
			0,
			0,
		)
		if hwnd == 0 {
			continue
		}

		created = append(created, windowsOverlayWindow{
			hwnd:    hwnd,
			x:       m.x,
			y:       m.y,
			w:       m.w,
			h:       m.h,
			primary: m.primary,
		})
	}

	if len(created) == 0 {
		c.ready <- false
		return
	}

	c.mu.Lock()
	c.windows = created
	c.started = true
	c.mu.Unlock()

	overlayWndProcMu.Lock()
	for _, wnd := range created {
		overlayByHwnd[wnd.hwnd] = c
	}
	overlayWndProcMu.Unlock()

	c.ready <- true
	for _, wnd := range created {
		postMessage(wnd.hwnd, msgOverlayApply, 0, 0)
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

func (c *windowsBreakOverlayController) apply(hwnd uintptr) {
	c.reconcileOverlayWindows()

	c.mu.RLock()
	visible := c.visible
	wnd, ok := c.lookupWindow(hwnd)
	c.mu.RUnlock()
	if !ok {
		return
	}

	x := wnd.x
	y := wnd.y
	w := wnd.w
	h := wnd.h
	if mx, my, mw, mh, ok := currentMonitorBoundsForWindow(hwnd); ok {
		x, y, w, h = mx, my, mw, mh
		c.mu.Lock()
		for i := range c.windows {
			if c.windows[i].hwnd == hwnd {
				c.windows[i].x = x
				c.windows[i].y = y
				c.windows[i].w = w
				c.windows[i].h = h
				wnd.primary = c.windows[i].primary
				break
			}
		}
		c.mu.Unlock()
	}

	shouldActivate := false
	c.mu.Lock()
	if visible && c.activationPending {
		if c.focusOwnerHwnd == 0 || !c.hasWindowLocked(c.focusOwnerHwnd) {
			c.focusOwnerHwnd = c.chooseFocusOwnerWindowLocked()
		}
		if c.focusOwnerHwnd == hwnd {
			shouldActivate = true
			c.activationPending = false
		}
	}
	c.mu.Unlock()

	_, _, _ = procSetWindowPos.Call(
		hwnd,
		hwndTopmost,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		overlayWindowActivationFlags(shouldActivate),
	)

	btnW := 260
	btnH := 58
	btnX := (w - btnW) / 2
	btnY := h/2 + 44
	allowSkip := c.allowSkip || c.emergencySkipVisible
	releaseCapture := false
	c.mu.Lock()
	for i := range c.windows {
		if c.windows[i].hwnd == hwnd {
			c.windows[i].buttonRect = ovlRect{
				left:   int32(btnX),
				top:    int32(btnY),
				right:  int32(btnX + btnW),
				bottom: int32(btnY + btnH),
			}
			if !allowSkip {
				if c.windows[i].buttonPressed {
					releaseCapture = true
				}
				c.windows[i].buttonHot = false
				c.windows[i].buttonPressed = false
				c.windows[i].trackingMouse = false
			}
			break
		}
	}
	c.mu.Unlock()
	if releaseCapture {
		_, _, _ = procReleaseCapture.Call()
	}
	ensureOverlayWindowLayered(hwnd)
	if visible {
		_, _, _ = procSetWindowPos.Call(
			hwnd,
			hwndTopmost,
			uintptr(x), uintptr(y), uintptr(w), uintptr(h),
			overlayWindowActivationFlags(shouldActivate),
		)
		if !wnd.shown {
			setOverlayWindowAlpha(hwnd, 0)
			_, _, _ = procSetWindowPos.Call(
				hwnd,
				hwndTopmost,
				uintptr(x), uintptr(y), uintptr(w), uintptr(h),
				swpShowWindow|overlayWindowActivationFlags(shouldActivate),
			)
			_, _, _ = procShowWindow.Call(hwnd, swShow)
			_, _, _ = procInvalidateRect.Call(hwnd, 0, 1)
			_, _, _ = procUpdateWindow.Call(hwnd)
			if shouldActivate {
				activateOverlayWindow(hwnd)
			}
			fadeOverlayWindowAlpha(hwnd, 0, 255, overlayFadeInDurationMs)
			c.mu.Lock()
			for i := range c.windows {
				if c.windows[i].hwnd == hwnd {
					c.windows[i].shown = true
					break
				}
			}
			c.mu.Unlock()
		} else {
			setOverlayWindowAlpha(hwnd, 255)
			_, _, _ = procSetWindowPos.Call(
				hwnd,
				hwndTopmost,
				uintptr(x), uintptr(y), uintptr(w), uintptr(h),
				swpShowWindow|overlayWindowActivationFlags(shouldActivate),
			)
			_, _, _ = procShowWindow.Call(hwnd, swShow)
			if shouldActivate {
				activateOverlayWindow(hwnd)
			}
		}
	} else {
		if wnd.shown {
			fadeOverlayWindowAlpha(hwnd, 255, 0, overlayFadeOutDurationMs)
			_, _, _ = procSetWindowPos.Call(
				hwnd,
				hwndTopmost,
				uintptr(x), uintptr(y), uintptr(w), uintptr(h),
				swpNoActivate|swpHideWindow|swpNoZOrder,
			)
			_, _, _ = procShowWindow.Call(hwnd, swHide)
			setOverlayWindowAlpha(hwnd, 255)
			c.mu.Lock()
			for i := range c.windows {
				if c.windows[i].hwnd == hwnd {
					if c.windows[i].buttonPressed {
						releaseCapture = true
					}
					c.windows[i].shown = false
					c.windows[i].buttonHot = false
					c.windows[i].buttonPressed = false
					c.windows[i].trackingMouse = false
					break
				}
			}
			c.mu.Unlock()
			if releaseCapture {
				_, _, _ = procReleaseCapture.Call()
			}
		} else {
			_, _, _ = procSetWindowPos.Call(
				hwnd,
				hwndTopmost,
				uintptr(x), uintptr(y), uintptr(w), uintptr(h),
				swpNoActivate|swpHideWindow|swpNoZOrder,
			)
			_, _, _ = procShowWindow.Call(hwnd, swHide)
		}
	}
	_, _, _ = procInvalidateRect.Call(hwnd, 0, 1)
}

func (c *windowsBreakOverlayController) handleSkipClick() {
	c.mu.RLock()
	if !c.allowSkip && !c.emergencySkipVisible {
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
	allowSkip := c.allowSkip || c.emergencySkipVisible
	skipText := fallbackOverlay(c.skipButtonTitle, "Emergency Skip")
	wnd, _ := c.lookupWindow(hwnd)
	c.mu.RUnlock()
	if strings.TrimSpace(text) == "" {
		text = ""
	}

	bg := colorBlack
	fg := colorWhite
	buttonBg := colorButtonDarkBg
	buttonFg := colorButtonDarkFg
	buttonAlpha := byte(alphaButtonBase)
	if theme == "light" {
		bg = colorWhite
		fg = colorBlack
		buttonBg = colorButtonLightBg
		buttonFg = colorButtonLightFg
	}
	if allowSkip {
		pressed := wnd.buttonPressed && wnd.buttonHot
		if theme == "light" {
			if pressed {
				buttonBg = colorButtonLightBgPressed
				buttonAlpha = byte(alphaButtonPressed)
			} else if wnd.buttonHot {
				buttonBg = colorButtonLightBgHover
				buttonAlpha = byte(alphaButtonHover)
			}
		} else {
			if pressed {
				buttonBg = colorButtonDarkBgPressed
				buttonAlpha = byte(alphaButtonPressed)
			} else if wnd.buttonHot {
				buttonBg = colorButtonDarkBgHover
				buttonAlpha = byte(alphaButtonHover)
			}
		}
	}

	var ps ovlPaintStruct
	hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}

	var client ovlRect
	_, _, _ = procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
	if client.right <= client.left || client.bottom <= client.top {
		client = ps.rcPaint
	}

	brush, _, _ := procCreateSolidBrush.Call(uintptr(bg))
	if brush != 0 {
		_, _, _ = procFillRect.Call(hdc, uintptr(unsafe.Pointer(&client)), brush)
		_, _, _ = procDeleteObject.Call(brush)
	}

	// TRANSPARENT background for text drawing.
	_, _, _ = procSetBkMode.Call(hdc, 1)
	_, _, _ = procSetTextColor.Call(hdc, uintptr(fg))

	textRect := ovlRect{
		left:   client.left,
		top:    client.top + 48,
		right:  client.right,
		bottom: client.bottom - 160,
	}
	utf16Text := syscall.StringToUTF16(text)
	if len(utf16Text) == 0 {
		utf16Text = []uint16{0}
	}
	countdownFont := createOverlayFont(-96, fwBold)
	if countdownFont != 0 {
		oldFont, _, _ := procSelectObject.Call(hdc, countdownFont)
		_, _, _ = procDrawTextW.Call(
			hdc,
			uintptr(unsafe.Pointer(&utf16Text[0])),
			^uintptr(0),
			uintptr(unsafe.Pointer(&textRect)),
			dtCenter|dtVCenter|dtSingleLine|dtNoPrefix,
		)
		_, _, _ = procSelectObject.Call(hdc, oldFont)
		_, _, _ = procDeleteObject.Call(countdownFont)
	} else {
		_, _, _ = procDrawTextW.Call(
			hdc,
			uintptr(unsafe.Pointer(&utf16Text[0])),
			^uintptr(0),
			uintptr(unsafe.Pointer(&textRect)),
			dtCenter|dtVCenter|dtSingleLine|dtNoPrefix,
		)
	}

	if allowSkip {
		btn := wnd.buttonRect
		blendedBg := blendOverlayColor(buttonBg, bg, buttonAlpha)
		outerBrush, _, _ := procCreateSolidBrush.Call(uintptr(blendedBg))
		if outerBrush != 0 {
			_, _, _ = procFillRect.Call(hdc, uintptr(unsafe.Pointer(&btn)), outerBrush)
			_, _, _ = procDeleteObject.Call(outerBrush)
		}
		_, _, _ = procSetTextColor.Call(hdc, uintptr(buttonFg))
		_, _, _ = procSetBkMode.Call(hdc, 1)
		btnText := syscall.StringToUTF16(skipText)
		if len(btnText) == 0 {
			btnText = []uint16{0}
		}
		buttonFont := createOverlayFont(-32, fwBold)
		if buttonFont != 0 {
			oldBtnFont, _, _ := procSelectObject.Call(hdc, buttonFont)
			_, _, _ = procDrawTextW.Call(
				hdc,
				uintptr(unsafe.Pointer(&btnText[0])),
				^uintptr(0),
				uintptr(unsafe.Pointer(&btn)),
				dtCenter|dtVCenter|dtSingleLine|dtNoPrefix,
			)
			_, _, _ = procSelectObject.Call(hdc, oldBtnFont)
			_, _, _ = procDeleteObject.Call(buttonFont)
		} else {
			_, _, _ = procDrawTextW.Call(
				hdc,
				uintptr(unsafe.Pointer(&btnText[0])),
				^uintptr(0),
				uintptr(unsafe.Pointer(&btn)),
				dtCenter|dtVCenter|dtSingleLine|dtNoPrefix,
			)
		}
	}
	_, _, _ = procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
}

func (c *windowsBreakOverlayController) destroy(hwnd uintptr) {
	overlayWndProcMu.Lock()
	delete(overlayByHwnd, hwnd)
	overlayWndProcMu.Unlock()

	c.mu.Lock()
	remaining := c.removeWindowLocked(hwnd)
	if c.focusOwnerHwnd == hwnd {
		c.focusOwnerHwnd = 0
	}
	if remaining == 0 {
		c.started = false
		c.windows = nil
		c.activationPending = false
	}
	c.mu.Unlock()
	if remaining == 0 {
		_, _, _ = procPostQuitMessage.Call(0)
	}
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
	case wmMouseMoveLocal:
		x := int(int16(uint16(lParam & 0xFFFF)))
		y := int(int16(uint16((lParam >> 16) & 0xFFFF)))
		if ctrl.handleMouseMove(hwnd, x, y) {
			return 0
		}
	case wmMouseLeaveLocal:
		if ctrl.handleMouseLeave(hwnd) {
			return 0
		}
	case wmLButtonDownLocal:
		x := int(int16(uint16(lParam & 0xFFFF)))
		y := int(int16(uint16((lParam >> 16) & 0xFFFF)))
		if ctrl.handleLButtonDown(hwnd, x, y) {
			return 0
		}
	case wmKeyDownLocal, wmSysKeyDown:
		if ctrl.handleBlockedShortcut(msgID, wParam) {
			return 0
		}
	case wmLButtonUpLocal:
		x := int(int16(uint16(lParam & 0xFFFF)))
		y := int(int16(uint16((lParam >> 16) & 0xFFFF)))
		if ctrl.handleLButtonUp(hwnd, x, y) {
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

func createOverlayFont(height int32, weight int32) uintptr {
	font, _, _ := procCreateFontW.Call(
		uintptr(height), // logical height (negative => character height)
		0,               // width
		0,               // escapement
		0,               // orientation
		uintptr(weight),
		0, // italic
		0, // underline
		0, // strikeout
		1, // DEFAULT_CHARSET
		0, // OUT_DEFAULT_PRECIS
		0, // CLIP_DEFAULT_PRECIS
		5, // CLEARTYPE_QUALITY
		0, // DEFAULT_PITCH | FF_DONTCARE
		0, // face name (system default UI font)
	)
	return font
}

func loadArrowCursor() uintptr {
	cur, _, _ := procLoadCursorW.Call(0, uintptr(idcArrow))
	return cur
}

func (c *windowsBreakOverlayController) hitSkipButton(hwnd uintptr, x int, y int) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.allowSkip && !c.emergencySkipVisible {
		return false
	}
	wnd, ok := c.lookupWindow(hwnd)
	if !ok {
		return false
	}
	r := wnd.buttonRect
	return pointInOverlayRect(r, x, y)
}

func pointInOverlayRect(r ovlRect, x int, y int) bool {
	return x >= int(r.left) && x < int(r.right) && y >= int(r.top) && y < int(r.bottom)
}

func trackOverlayMouseLeave(hwnd uintptr) {
	tme := ovlTrackMouseEvent{
		cbSize:    uint32(unsafe.Sizeof(ovlTrackMouseEvent{})),
		dwFlags:   tmeLeave,
		hwndTrack: hwnd,
	}
	_, _, _ = procTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
}

func ensureOverlayWindowLayered(hwnd uintptr) {
	exStyle, _, _ := procGetWindowLongPtrWOvl.Call(hwnd, gwlExStyleIdx)
	if exStyle&wsExLayered != 0 {
		return
	}
	_, _, _ = procSetWindowLongPtrWOvl.Call(hwnd, gwlExStyleIdx, exStyle|wsExLayered)
}

func setOverlayWindowAlpha(hwnd uintptr, alpha byte) {
	_, _, _ = procSetLayeredWindowAttributes.Call(hwnd, 0, uintptr(alpha), lwaAlpha)
}

func fadeOverlayWindowAlpha(hwnd uintptr, from byte, to byte, durationMs int) {
	if durationMs <= 0 || from == to {
		setOverlayWindowAlpha(hwnd, to)
		return
	}

	steps := durationMs / overlayFadeStepMs
	if steps < 1 {
		steps = 1
	}
	stepDuration := time.Duration(durationMs/steps) * time.Millisecond
	delta := int(to) - int(from)
	for i := 0; i <= steps; i++ {
		// Keep 1s duration but make fade-in perceptible from the first frames.
		t := float64(i) / float64(steps)
		var eased float64
		if delta > 0 {
			// Ease-out for fade-in: appears quickly instead of looking like a 1s delay.
			eased = 1 - (1-t)*(1-t)
		} else {
			// Ease-in for fade-out: leaves a cleaner tail when exiting.
			eased = t * t
		}
		alpha := int(from) + int(float64(delta)*eased)
		if alpha < 0 {
			alpha = 0
		}
		if alpha > 255 {
			alpha = 255
		}
		setOverlayWindowAlpha(hwnd, byte(alpha))
		if i < steps {
			time.Sleep(stepDuration)
		}
	}
}

func (c *windowsBreakOverlayController) handleMouseMove(hwnd uintptr, x int, y int) bool {
	needsInvalidate := false
	needTrack := false

	c.mu.Lock()
	allowSkip := c.allowSkip || c.emergencySkipVisible
	for i := range c.windows {
		if c.windows[i].hwnd != hwnd {
			continue
		}
		hot := allowSkip && pointInOverlayRect(c.windows[i].buttonRect, x, y)
		if c.windows[i].buttonHot != hot {
			c.windows[i].buttonHot = hot
			needsInvalidate = true
		}
		if !c.windows[i].trackingMouse {
			c.windows[i].trackingMouse = true
			needTrack = true
		}
		break
	}
	c.mu.Unlock()

	if needTrack {
		trackOverlayMouseLeave(hwnd)
	}
	if needsInvalidate {
		_, _, _ = procInvalidateRect.Call(hwnd, 0, 1)
	}
	return false
}

func (c *windowsBreakOverlayController) handleMouseLeave(hwnd uintptr) bool {
	needsInvalidate := false

	c.mu.Lock()
	for i := range c.windows {
		if c.windows[i].hwnd != hwnd {
			continue
		}
		if c.windows[i].buttonHot {
			c.windows[i].buttonHot = false
			needsInvalidate = true
		}
		c.windows[i].trackingMouse = false
		break
	}
	c.mu.Unlock()

	if needsInvalidate {
		_, _, _ = procInvalidateRect.Call(hwnd, 0, 1)
	}
	return false
}

func (c *windowsBreakOverlayController) handleLButtonDown(hwnd uintptr, x int, y int) bool {
	handled := false
	needsInvalidate := false
	needTrack := false

	c.mu.Lock()
	allowSkip := c.allowSkip || c.emergencySkipVisible
	for i := range c.windows {
		if c.windows[i].hwnd != hwnd {
			continue
		}
		if allowSkip && pointInOverlayRect(c.windows[i].buttonRect, x, y) {
			handled = true
			if !c.windows[i].buttonPressed {
				c.windows[i].buttonPressed = true
				needsInvalidate = true
			}
			if !c.windows[i].buttonHot {
				c.windows[i].buttonHot = true
				needsInvalidate = true
			}
			if !c.windows[i].trackingMouse {
				c.windows[i].trackingMouse = true
				needTrack = true
			}
		}
		break
	}
	c.mu.Unlock()

	if !handled {
		return false
	}
	_, _, _ = procSetCapture.Call(hwnd)
	if needTrack {
		trackOverlayMouseLeave(hwnd)
	}
	if needsInvalidate {
		_, _, _ = procInvalidateRect.Call(hwnd, 0, 1)
	}
	return true
}

func (c *windowsBreakOverlayController) handleLButtonUp(hwnd uintptr, x int, y int) bool {
	trigger := false
	wasPressed := false
	needsInvalidate := false

	c.mu.Lock()
	allowSkip := c.allowSkip || c.emergencySkipVisible
	for i := range c.windows {
		if c.windows[i].hwnd != hwnd {
			continue
		}
		hot := allowSkip && pointInOverlayRect(c.windows[i].buttonRect, x, y)
		if c.windows[i].buttonHot != hot {
			c.windows[i].buttonHot = hot
			needsInvalidate = true
		}
		if c.windows[i].buttonPressed {
			wasPressed = true
			c.windows[i].buttonPressed = false
			needsInvalidate = true
			if hot && allowSkip {
				trigger = true
			}
		}
		break
	}
	c.mu.Unlock()

	if wasPressed {
		_, _, _ = procReleaseCapture.Call()
	}
	if needsInvalidate {
		_, _, _ = procInvalidateRect.Call(hwnd, 0, 1)
	}
	return trigger
}

func (c *windowsBreakOverlayController) handleBlockedShortcut(msgID uint32, wParam uintptr) bool {
	if !isOverlayBlockedShortcut(msgID, wParam) {
		return false
	}

	c.mu.Lock()
	allowSkip := c.allowSkip
	if allowSkip {
		c.mu.Unlock()
		return true
	}

	now := time.Now()
	if c.lastBlockedShortcutAt.IsZero() || now.Sub(c.lastBlockedShortcutAt) > 1200*time.Millisecond {
		c.blockedShortcutCount = 1
	} else {
		c.blockedShortcutCount++
	}
	c.lastBlockedShortcutAt = now

	shouldReveal := c.blockedShortcutCount >= 2 && !c.emergencySkipVisible
	if shouldReveal {
		c.emergencySkipVisible = true
		c.blockedShortcutCount = 0
	}
	windows := append([]windowsOverlayWindow(nil), c.windows...)
	c.mu.Unlock()

	if shouldReveal {
		for _, wnd := range windows {
			if wnd.hwnd != 0 {
				_, _, _ = procInvalidateRect.Call(wnd.hwnd, 0, 1)
			}
		}
	}
	return true
}

func isOverlayBlockedShortcut(msgID uint32, wParam uintptr) bool {
	vk := int(wParam)
	switch vk {
	case vkEscape:
		return true
	case vkF4:
		return msgID == wmSysKeyDown && isVirtualKeyDown(vkMenu)
	case vkSpace:
		return msgID == wmSysKeyDown && isVirtualKeyDown(vkMenu)
	case vkW:
		return msgID == wmKeyDownLocal && isVirtualKeyDown(vkControl)
	default:
		return false
	}
}

func isVirtualKeyDown(vk int) bool {
	v, _, _ := procGetKeyState.Call(uintptr(vk))
	return (v & 0x8000) != 0
}

func blendOverlayColor(fg int, bg int, alpha byte) int {
	if alpha == 255 {
		return fg
	}
	a := int(alpha)
	inv := 255 - a
	r := (((fg>>16)&0xFF)*a + ((bg>>16)&0xFF)*inv) / 255
	g := (((fg>>8)&0xFF)*a + ((bg>>8)&0xFF)*inv) / 255
	b := ((fg&0xFF)*a + (bg&0xFF)*inv) / 255
	return (r << 16) | (g << 8) | b
}

func (c *windowsBreakOverlayController) reconcileOverlayWindows() {
	monitors := listOverlayMonitors()
	if len(monitors) == 0 {
		return
	}

	c.mu.RLock()
	className := c.className
	existing := append([]windowsOverlayWindow(nil), c.windows...)
	c.mu.RUnlock()
	if strings.TrimSpace(className) == "" {
		return
	}

	wanted := make(map[string]ovlMonitorBounds, len(monitors))
	for _, m := range monitors {
		wanted[overlayMonitorKey(m.x, m.y, m.w, m.h)] = m
	}
	existingByKey := make(map[string]windowsOverlayWindow, len(existing))
	for _, wnd := range existing {
		existingByKey[overlayMonitorKey(wnd.x, wnd.y, wnd.w, wnd.h)] = wnd
	}

	toAdd := make([]ovlMonitorBounds, 0)
	for k, m := range wanted {
		if _, ok := existingByKey[k]; !ok {
			toAdd = append(toAdd, m)
		}
	}
	toRemove := make([]windowsOverlayWindow, 0)
	for k, wnd := range existingByKey {
		if _, ok := wanted[k]; !ok {
			toRemove = append(toRemove, wnd)
		}
	}

	if len(toAdd) == 0 && len(toRemove) == 0 {
		return
	}

	for _, m := range toAdd {
		if wnd, ok := c.createOverlayWindowForMonitor(className, m); ok {
			c.mu.Lock()
			c.windows = append(c.windows, wnd)
			c.mu.Unlock()
			overlayWndProcMu.Lock()
			overlayByHwnd[wnd.hwnd] = c
			overlayWndProcMu.Unlock()
			postMessage(wnd.hwnd, msgOverlayApply, 0, 0)
		}
	}
	for _, wnd := range toRemove {
		_, _, _ = procDestroyWindow.Call(wnd.hwnd)
	}
}

func (c *windowsBreakOverlayController) createOverlayWindowForMonitor(className string, m ovlMonitorBounds) (windowsOverlayWindow, bool) {
	classNamePtr, err := syscall.UTF16PtrFromString(className)
	if err != nil {
		return windowsOverlayWindow{}, false
	}
	windowNamePtr, _ := syscall.UTF16PtrFromString("PauseOverlay")
	hwnd, _, _ := procCreateWindowExW.Call(
		wsExTopmost|wsExToolWindow,
		uintptr(unsafe.Pointer(classNamePtr)),
		uintptr(unsafe.Pointer(windowNamePtr)),
		wsPopup,
		uintptr(m.x), uintptr(m.y), uintptr(m.w), uintptr(m.h),
		0, 0, 0, 0,
	)
	if hwnd == 0 {
		return windowsOverlayWindow{}, false
	}
	return windowsOverlayWindow{
		hwnd:    hwnd,
		x:       m.x,
		y:       m.y,
		w:       m.w,
		h:       m.h,
		primary: m.primary,
	}, true
}

func overlayMonitorKey(x, y, w, h int) string {
	return fmt.Sprintf("%d:%d:%d:%d", x, y, w, h)
}

func (c *windowsBreakOverlayController) lookupWindow(hwnd uintptr) (windowsOverlayWindow, bool) {
	for _, wnd := range c.windows {
		if wnd.hwnd == hwnd {
			return wnd, true
		}
	}
	return windowsOverlayWindow{}, false
}

func (c *windowsBreakOverlayController) hasWindowLocked(hwnd uintptr) bool {
	for _, wnd := range c.windows {
		if wnd.hwnd == hwnd {
			return true
		}
	}
	return false
}

func (c *windowsBreakOverlayController) chooseFocusOwnerWindowLocked() uintptr {
	if len(c.windows) == 0 {
		return 0
	}

	var cursor point
	if ret, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&cursor))); ret != 0 {
		x := int(cursor.x)
		y := int(cursor.y)
		for _, wnd := range c.windows {
			if x >= wnd.x && x < wnd.x+wnd.w && y >= wnd.y && y < wnd.y+wnd.h {
				return wnd.hwnd
			}
		}
	}

	for _, wnd := range c.windows {
		if wnd.primary {
			return wnd.hwnd
		}
	}
	return c.windows[0].hwnd
}

func (c *windowsBreakOverlayController) removeWindowLocked(hwnd uintptr) int {
	out := c.windows[:0]
	for _, wnd := range c.windows {
		if wnd.hwnd != hwnd {
			out = append(out, wnd)
		}
	}
	c.windows = out
	return len(c.windows)
}

func listOverlayMonitors() []ovlMonitorBounds {
	result := make([]ovlMonitorBounds, 0, 4)

	overlayMonitorEnumMu.Lock()
	overlayMonitorsOut = &result
	overlayMonitorEnumMu.Unlock()

	_, _, _ = procEnumDisplayMonitors.Call(0, 0, overlayMonitorEnumCb, 0)

	overlayMonitorEnumMu.Lock()
	overlayMonitorsOut = nil
	overlayMonitorEnumMu.Unlock()

	if len(result) == 0 {
		x := getSystemMetric(smXVirtualScreen)
		y := getSystemMetric(smYVirtualScreen)
		w := getSystemMetric(smCXVirtualScreen)
		h := getSystemMetric(smCYVirtualScreen)
		if w <= 0 || h <= 0 {
			w = 1920
			h = 1080
		}
		result = append(result, ovlMonitorBounds{x: x, y: y, w: w, h: h, primary: true})
	}
	return result
}

func windowsMonitorEnumProc(hMonitor uintptr, _ uintptr, _ uintptr, _ uintptr) uintptr {
	overlayMonitorEnumMu.Lock()
	out := overlayMonitorsOut
	overlayMonitorEnumMu.Unlock()
	if out == nil {
		return 1
	}

	info := ovlMonitorInfo{
		cbSize: uint32(unsafe.Sizeof(ovlMonitorInfo{})),
	}
	ret, _, _ := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&info)))
	if ret == 0 {
		return 1
	}
	w := int(info.rcMonitor.right - info.rcMonitor.left)
	h := int(info.rcMonitor.bottom - info.rcMonitor.top)
	if w <= 0 || h <= 0 {
		return 1
	}
	*out = append(*out, ovlMonitorBounds{
		x:       int(info.rcMonitor.left),
		y:       int(info.rcMonitor.top),
		w:       w,
		h:       h,
		primary: info.dwFlags&1 != 0,
	})
	return 1
}

func overlayWindowActivationFlags(activate bool) uintptr {
	if activate {
		return 0
	}
	return swpNoActivate
}

func activateOverlayWindow(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	_, _, _ = procSetForegroundWnd.Call(hwnd)
	_, _, _ = procSetActiveWindow.Call(hwnd)
	_, _, _ = procSetFocus.Call(hwnd)
}

func currentMonitorBoundsForWindow(hwnd uintptr) (int, int, int, int, bool) {
	if hwnd == 0 {
		return 0, 0, 0, 0, false
	}
	hMonitor, _, _ := procMonitorFromWindow.Call(hwnd, monitorDefaultToNearest)
	if hMonitor == 0 {
		return 0, 0, 0, 0, false
	}
	info := ovlMonitorInfo{
		cbSize: uint32(unsafe.Sizeof(ovlMonitorInfo{})),
	}
	ret, _, _ := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&info)))
	if ret == 0 {
		return 0, 0, 0, 0, false
	}
	x := int(info.rcMonitor.left)
	y := int(info.rcMonitor.top)
	w := int(info.rcMonitor.right - info.rcMonitor.left)
	h := int(info.rcMonitor.bottom - info.rcMonitor.top)
	if w <= 0 || h <= 0 {
		return 0, 0, 0, 0, false
	}
	return x, y, w, h, true
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
