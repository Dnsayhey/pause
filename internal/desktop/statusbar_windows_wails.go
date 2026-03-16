//go:build wails && windows

package desktop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

	imageIcon      = 1
	lrLoadFromFile = 0x0010
)

const (
	menuBreakNow = 101
	menuPause    = 102
	menuResume   = 104
	menuOpen     = 105
	menuQuit     = 107
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
	procLoadImageW       = user32DLL.NewProc("LoadImageW")

	procShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")
)

type windowsStatusBarController struct {
	mu sync.RWMutex

	onEvent func(StatusBarEvent)
	locale  StatusBarLocaleStrings

	status    string
	countdown string
	title     string
	paused    bool
	canBreak  bool
	reminders []windowsReminderStatusView

	hwnd      uintptr
	startOnce sync.Once
	done      chan struct{}
}

type windowsReminderStatusView struct {
	Reason string `json:"reason"`
	Paused bool   `json:"paused"`
	Title  string `json:"title"`
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

func (c *windowsStatusBarController) Init(onEvent func(event StatusBarEvent)) {
	c.mu.Lock()
	c.onEvent = onEvent
	c.mu.Unlock()

	c.startOnce.Do(func() {
		go c.loop()
	})
}

func (c *windowsStatusBarController) Update(status, countdown, title string, paused bool, _ float64, remindersPayload string) {
	items, parsed := parseReminderItems(remindersPayload)
	hasPayload := strings.TrimSpace(remindersPayload) != ""

	c.mu.Lock()
	c.status = strings.TrimSpace(status)
	c.countdown = flattenMenuText(countdown)
	c.title = strings.TrimSpace(title)
	c.paused = paused
	c.reminders = append([]windowsReminderStatusView(nil), items...)
	c.canBreak = canTriggerBreakNow(paused, hasPayload, parsed, items)
	hwnd := c.hwnd
	c.mu.Unlock()

	if hwnd != 0 {
		postMessage(hwnd, msgTrayUpdate, 0, 0)
	}
}

func flattenMenuText(text string) string {
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return r == '\r' || r == '\n'
	})
	if len(parts) == 0 {
		return ""
	}
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		cleaned = append(cleaned, part)
	}
	return strings.Join(cleaned, " | ")
}

func parseReminderItems(remindersPayload string) ([]windowsReminderStatusView, bool) {
	trimmed := strings.TrimSpace(remindersPayload)
	if trimmed == "" {
		return nil, true
	}

	var items []windowsReminderStatusView
	if err := json.Unmarshal([]byte(trimmed), &items); err != nil {
		return nil, false
	}
	return items, true
}

func canTriggerBreakNow(paused bool, hasPayload bool, parsed bool, items []windowsReminderStatusView) bool {
	if paused {
		return false
	}
	if !parsed {
		// Keep break-now available if payload format changes but is non-empty.
		return hasPayload
	}
	for _, item := range items {
		if !item.Paused {
			return true
		}
	}
	return false
}

func isLikelyChinese(locale StatusBarLocaleStrings) bool {
	sample := locale.PauseButton + locale.ResumeButton + locale.BreakNowButton
	for _, r := range sample {
		if r > 127 {
			return true
		}
	}
	return false
}

func reminderStateLabel(locale StatusBarLocaleStrings, paused bool) string {
	if isLikelyChinese(locale) {
		if paused {
			return "暂停中"
		}
		return "运行中"
	}
	if paused {
		return "Paused"
	}
	return "Running"
}

func reminderActionLabel(locale StatusBarLocaleStrings, paused bool) string {
	if isLikelyChinese(locale) {
		if paused {
			return "恢复"
		}
		return "暂停"
	}
	if paused {
		return "Resume"
	}
	return "Pause"
}

func (c *windowsStatusBarController) SetLocale(strings StatusBarLocaleStrings) {
	c.mu.Lock()
	c.locale = strings
	hwnd := c.hwnd
	c.mu.Unlock()
	if hwnd != 0 {
		postMessage(hwnd, msgTrayUpdate, 0, 0)
	}
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
	icon := loadTrayIcon()
	if icon == 0 {
		icon, _, _ = procLoadIconW.Call(0, uintptr(idiApplication))
	}
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

func loadTrayIcon() uintptr {
	// Only accept icon.ico next to the executable.
	exePath, err := os.Executable()
	if err != nil || strings.TrimSpace(exePath) == "" {
		return 0
	}
	iconPath := filepath.Join(filepath.Dir(exePath), "icon.ico")
	if _, err := os.Stat(iconPath); err != nil {
		return 0
	}
	ptr, err := syscall.UTF16PtrFromString(iconPath)
	if err != nil {
		return 0
	}
	icon, _, _ := procLoadImageW.Call(
		0,
		uintptr(unsafe.Pointer(ptr)),
		imageIcon,
		0,
		0,
		lrLoadFromFile,
	)
	if icon == 0 {
		return 0
	}
	return icon
}

func (c *windowsStatusBarController) updateTooltip(hwnd uintptr) {
	c.mu.RLock()
	tip := composeTrayTip(c.status, c.countdown, c.title, c.locale.Tooltip)
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

func composeTrayTip(status, countdown, title, fallbackTip string) string {
	parts := make([]string, 0, 3)
	if status != "" {
		parts = append(parts, status)
	}
	if countdown != "" {
		parts = append(parts, countdown)
	}
	// The countdown payload already contains per-reminder remaining time.
	// Avoid appending an extra standalone countdown (title), which appears duplicated in tooltip.
	_ = title
	if len(parts) == 0 {
		return fallback(fallbackTip, "Pause")
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
	canBreak := c.canBreak
	countdown := c.countdown
	reminders := append([]windowsReminderStatusView(nil), c.reminders...)
	locale := c.locale
	c.mu.RUnlock()
	countdownLine := strings.TrimSpace(countdown)
	if countdownLine == "" {
		countdownLine = fallback(locale.NextBreakLineFallback, "Next break: --:--")
	}

	labelBreakNow := fallback(locale.BreakNowButton, "Break now")
	labelToggle := fallback(locale.PauseButton, "Pause")
	if paused {
		labelToggle = fallback(locale.ResumeButton, "Resume")
	}
	labelOpen := fallback(locale.OpenAppButton, "Open")
	labelQuit := fallback(locale.QuitMenuItem, "Quit")

	if len(reminders) == 0 {
		addMenuText(menu, 0, countdownLine, true)
	} else {
		for idx, reminder := range reminders {
			title := strings.TrimSpace(reminder.Title)
			if title == "" {
				title = fallback(reminder.Reason, "Reminder")
			}
			headline := fmt.Sprintf("%s - %s", reminderStateLabel(locale, reminder.Paused), title)
			addMenuText(menu, 0, headline, true)

			actionLabel := "  " + reminderActionLabel(locale, reminder.Paused)
			actionID := StatusBarActionPauseReminderBase + idx
			if reminder.Paused {
				actionID = StatusBarActionResumeReminderBase + idx
			}
			addMenuText(menu, actionID, actionLabel, false)
		}
	}

	addMenuSeparator(menu)
	if paused {
		addMenuText(menu, menuResume, labelToggle, false)
	} else {
		addMenuText(menu, menuPause, labelToggle, false)
	}
	addMenuText(menu, menuBreakNow, labelBreakNow, !canBreak)
	addMenuSeparator(menu)
	addMenuText(menu, menuOpen, labelOpen, false)
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
	case menuResume:
		action = StatusBarActionResume
	case menuOpen:
		action = StatusBarActionOpenWindow
	case menuQuit:
		action = StatusBarActionQuit
	default:
		if id >= StatusBarActionPauseReminderBase && id < StatusBarActionPauseReminderBase+1000 {
			action = id
			break
		}
		if id >= StatusBarActionResumeReminderBase && id < StatusBarActionResumeReminderBase+1000 {
			action = id
			break
		}
		return
	}

	c.mu.RLock()
	cb := c.onEvent
	c.mu.RUnlock()
	if cb != nil {
		cb(StatusBarEvent{
			Kind:     StatusBarEventAction,
			ActionID: action,
		})
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
		case wmLButtonUp, wmRButtonUp, wmContext:
			ctrl.showContextMenu(hwnd)
		case wmLButtonDbl:
			ctrl.dispatchMenuCommand(menuOpen)
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
