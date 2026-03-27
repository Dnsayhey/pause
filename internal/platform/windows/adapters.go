//go:build windows

package windows

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	xwindows "golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"pause/internal/backend/domain/settings"
	"pause/internal/backend/ports"
	"pause/internal/meta"
	"pause/internal/platform/api"
)

const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
const appIDRegPathPrefix = `Software\Classes\AppUserModelId\`

var (
	user32DLL                                   = syscall.NewLazyDLL("user32.dll")
	kernel32DLL                                 = syscall.NewLazyDLL("kernel32.dll")
	shell32DLL                                  = syscall.NewLazyDLL("shell32.dll")
	procGetLastInputInfo                        = user32DLL.NewProc("GetLastInputInfo")
	procGetTickCount64                          = kernel32DLL.NewProc("GetTickCount64")
	procMessageBeep                             = user32DLL.NewProc("MessageBeep")
	procCreateWindowExW                         = user32DLL.NewProc("CreateWindowExW")
	procDestroyWindow                           = user32DLL.NewProc("DestroyWindow")
	procShellNotifyIconW                        = shell32DLL.NewProc("Shell_NotifyIconW")
	procShellExecuteW                           = shell32DLL.NewProc("ShellExecuteW")
	procSetCurrentProcessExplicitAppUserModelID = shell32DLL.NewProc("SetCurrentProcessExplicitAppUserModelID")
	execCommand                                 = exec.Command
	showToastReminder                           = showWinRTToastReminder
	showBalloonNotification                     = showBalloonReminder
	queryToastSetting                           = queryWindowsToastSettingNative
	openNotificationSettings                    = openWindowsNotificationSettingsNative
	balloonNotifyMu                             sync.Mutex
	balloonActiveHWND                           uintptr
)

const (
	nimAdd    = 0x00000000
	nimModify = 0x00000001
	nimDelete = 0x00000002

	nifTip  = 0x00000004
	nifInfo = 0x00000010
	nifGUID = 0x00000020

	niifInfo = 0x00000001

	mbIconAsterisk = 0x00000040

	hwndMessage = ^uintptr(2) // (HWND)-3
)

var pauseNotifyGUID = xwindows.GUID{
	Data1: 0x23a58e31,
	Data2: 0x8d36,
	Data3: 0x43f2,
	Data4: [8]byte{0x9c, 0x11, 0x4c, 0x6b, 0xfe, 0xc7, 0x09, 0xb4},
}

type windowsIdleProvider struct{}

type windowsNotifier struct {
	appID string
}

type windowsSoundPlayer struct{}

type windowsStartupManager struct {
	valueName string
}

type lastInputInfo struct {
	cbSize uint32
	dwTime uint32
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
	guidItem          xwindows.GUID
	hBalloonIcon      uintptr
}

func NewAdapters(appID string) api.Adapters {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		appID = meta.EffectiveAppBundleID()
	}
	toastID := toastAppID(appID)
	_ = setCurrentProcessAppUserModelID(toastID)
	return api.Adapters{
		IdleProvider:                   windowsIdleProvider{},
		LockStateProvider:              newWindowsLockStateProvider(),
		Notifier:                       windowsNotifier{appID: toastID},
		NotificationCapabilityProvider: windowsNotifier{appID: toastID},
		SoundPlayer:                    windowsSoundPlayer{},
		StartupManager:                 windowsStartupManager{valueName: startupValueName(appID)},
	}
}

func (windowsIdleProvider) CurrentIdleSeconds() int {
	info := lastInputInfo{cbSize: uint32(unsafe.Sizeof(lastInputInfo{}))}
	ret, _, _ := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&info)))
	if ret == 0 {
		return 0
	}

	tick64, _, _ := procGetTickCount64.Call()
	if tick64 == 0 {
		return 0
	}

	now := uint64(tick64)
	last := uint64(info.dwTime)
	const wrap32 = uint64(^uint32(0)) + 1
	if now < last {
		now += wrap32
	}
	return int((now - last) / 1000)
}

func (n windowsNotifier) ShowReminder(title, body string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Pause"
	}
	body = strings.TrimSpace(body)
	if body == "" {
		body = "Break started"
	}

	appID := toastAppID(n.appID)
	if err := showToastReminder(appID, title, body); err == nil {
		return nil
	}
	return showBalloonNotification(title, body)
}

func (n windowsNotifier) GetNotificationCapability() ports.NotificationCapability {
	setting, err := queryToastSetting(toastAppID(n.appID))
	if err != nil {
		return ports.NotificationCapability{
			PermissionState: ports.NotificationPermissionUnknown,
			CanRequest:      false,
			CanOpenSettings: true,
			Reason:          err.Error(),
		}
	}
	switch setting {
	case "Enabled":
		return ports.NotificationCapability{
			PermissionState: ports.NotificationPermissionAuthorized,
			CanRequest:      false,
			CanOpenSettings: true,
		}
	case "DisabledForApplication":
		return ports.NotificationCapability{
			PermissionState: ports.NotificationPermissionDenied,
			CanRequest:      false,
			CanOpenSettings: true,
			Reason:          "notifications are disabled for this app",
		}
	case "DisabledForUser":
		return ports.NotificationCapability{
			PermissionState: ports.NotificationPermissionDenied,
			CanRequest:      false,
			CanOpenSettings: true,
			Reason:          "notifications are disabled for the current user",
		}
	case "DisabledByGroupPolicy":
		return ports.NotificationCapability{
			PermissionState: ports.NotificationPermissionRestricted,
			CanRequest:      false,
			CanOpenSettings: true,
			Reason:          "notifications are disabled by group policy",
		}
	case "DisabledForManifest":
		return ports.NotificationCapability{
			PermissionState: ports.NotificationPermissionRestricted,
			CanRequest:      false,
			CanOpenSettings: true,
			Reason:          "notifications are disabled by manifest configuration",
		}
	default:
		return ports.NotificationCapability{
			PermissionState: ports.NotificationPermissionUnknown,
			CanRequest:      false,
			CanOpenSettings: true,
			Reason:          "unknown toast notification setting: " + setting,
		}
	}
}

func (n windowsNotifier) RequestNotificationPermission() (ports.NotificationCapability, error) {
	return n.GetNotificationCapability(), nil
}

func (n windowsNotifier) OpenNotificationSettings() error {
	return openNotificationSettings()
}

func showWinRTToastReminder(appID, title, body string) error {
	if err := ensureToastAppIDRegistration(appID); err != nil {
		return err
	}

	// Standard Windows 10/11 notification path via WinRT Toast APIs.
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop';
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null;
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null;
$title = [Security.SecurityElement]::Escape('%s');
$body = [Security.SecurityElement]::Escape('%s');
$xml = @"
<toast>
  <visual>
    <binding template='ToastGeneric'>
      <text>$title</text>
      <text>$body</text>
    </binding>
  </visual>
</toast>
"@;
$doc = New-Object Windows.Data.Xml.Dom.XmlDocument;
$doc.LoadXml($xml);
$toast = [Windows.UI.Notifications.ToastNotification]::new($doc);
$notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('%s');
$notifier.Show($toast);
`,
		escapePowerShellSingleQuoted(title),
		escapePowerShellSingleQuoted(body),
		escapePowerShellSingleQuoted(appID),
	)
	return runPowerShellSync(script)
}

func showBalloonReminder(title, body string) error {
	balloonNotifyMu.Lock()
	defer balloonNotifyMu.Unlock()

	if balloonActiveHWND != 0 {
		cleanupBalloonIconLocked(balloonActiveHWND)
		balloonActiveHWND = 0
	}

	className, err := syscall.UTF16PtrFromString("STATIC")
	if err != nil {
		return err
	}
	windowTitle, err := syscall.UTF16PtrFromString("pause-notify")
	if err != nil {
		return err
	}
	hwnd, _, createErr := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowTitle)),
		0,
		0, 0, 0, 0,
		hwndMessage,
		0,
		0,
		0,
	)
	if hwnd == 0 {
		if createErr != syscall.Errno(0) {
			return createErr
		}
		return errors.New("CreateWindowExW returned null HWND")
	}

	add := notifyIconData{
		cbSize:   uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:     hwnd,
		uID:      1,
		uFlags:   nifTip | nifGUID,
		guidItem: pauseNotifyGUID,
	}
	copyUTF16(add.szTip[:], "Pause")

	if !shellNotifyIcon(nimAdd, &add) {
		_, _, _ = procDestroyWindow.Call(hwnd)
		return errors.New("Shell_NotifyIconW(NIM_ADD) failed")
	}

	mod := notifyIconData{
		cbSize:      uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:        hwnd,
		uID:         1,
		uFlags:      nifInfo | nifGUID,
		dwInfoFlags: niifInfo,
		guidItem:    pauseNotifyGUID,
	}
	copyUTF16(mod.szInfoTitle[:], title)
	copyUTF16(mod.szInfo[:], body)

	if !shellNotifyIcon(nimModify, &mod) {
		del := notifyIconData{
			cbSize:   uint32(unsafe.Sizeof(notifyIconData{})),
			hWnd:     hwnd,
			uID:      1,
			uFlags:   nifGUID,
			guidItem: pauseNotifyGUID,
		}
		shellNotifyIcon(nimDelete, &del)
		_, _, _ = procDestroyWindow.Call(hwnd)
		return errors.New("Shell_NotifyIconW(NIM_MODIFY) failed")
	}

	go func(targetHWND uintptr) {
		time.Sleep(7 * time.Second)
		balloonNotifyMu.Lock()
		defer balloonNotifyMu.Unlock()
		if balloonActiveHWND != targetHWND {
			return
		}
		cleanupBalloonIconLocked(targetHWND)
		balloonActiveHWND = 0
	}(hwnd)
	balloonActiveHWND = hwnd
	return nil
}

func (windowsSoundPlayer) PlayBreakEnd(sound settings.SoundSettings) error {
	if !sound.Enabled {
		return nil
	}
	ret, _, callErr := procMessageBeep.Call(uintptr(mbIconAsterisk))
	if ret != 0 {
		return nil
	}
	if callErr != syscall.Errno(0) {
		return callErr
	}
	return errors.New("MessageBeep returned 0")
}

func (s windowsStartupManager) SetLaunchAtLogin(enabled bool) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	if !enabled {
		err = key.DeleteValue(s.valueName)
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return err
	}

	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	if strings.TrimSpace(execPath) == "" {
		return errors.New("empty executable path")
	}
	if !filepath.IsAbs(execPath) {
		execPath, err = filepath.Abs(execPath)
		if err != nil {
			return err
		}
	}

	return key.SetStringValue(s.valueName, quoteCommandPath(execPath))
}

func (s windowsStartupManager) GetLaunchAtLogin() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer key.Close()

	raw, _, err := key.GetStringValue(s.valueName)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(raw) != "", nil
}

func startupValueName(appID string) string {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return "Pause"
	}
	appID = strings.ReplaceAll(appID, "/", ".")
	return appID
}

func quoteCommandPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "\"") && strings.HasSuffix(path, "\"") {
		return path
	}
	return `"` + path + `"`
}

func escapePowerShellSingleQuoted(value string) string {
	return strings.ReplaceAll(value, `'`, `''`)
}

func runPowerShellCommand(script string) *exec.Cmd {
	return execCommand("powershell.exe",
		"-NoProfile",
		"-NonInteractive",
		"-ExecutionPolicy",
		"Bypass",
		"-WindowStyle",
		"Hidden",
		"-STA",
		"-Command",
		script,
	)
}

func runPowerShellSync(script string) error {
	cmd := runPowerShellCommand(script)
	return cmd.Run()
}

func runPowerShellOutput(script string) (string, error) {
	cmd := runPowerShellCommand(script)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func queryWindowsToastSetting(appID string) (string, error) {
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop';
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null;
$notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('%s');
[Console]::WriteLine($notifier.Setting.ToString());
`,
		escapePowerShellSingleQuoted(appID),
	)
	return runPowerShellOutput(script)
}

func toastAppID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "com.pause.app"
	}
	raw = strings.ReplaceAll(raw, "/", ".")
	return raw
}

func ensureToastAppIDRegistration(appID string) error {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return errors.New("empty toast app id")
	}
	key, _, err := registry.CreateKey(registry.CURRENT_USER, appIDRegPathPrefix+appID, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	if err := key.SetStringValue("DisplayName", "Pause"); err != nil {
		return err
	}
	if execPath, err := os.Executable(); err == nil && strings.TrimSpace(execPath) != "" {
		_ = key.SetStringValue("IconUri", execPath)
	}
	return nil
}

func setCurrentProcessAppUserModelID(appID string) error {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return errors.New("empty app user model id")
	}
	ptr, err := syscall.UTF16PtrFromString(appID)
	if err != nil {
		return err
	}
	hr, _, _ := procSetCurrentProcessExplicitAppUserModelID.Call(uintptr(unsafe.Pointer(ptr)))
	if hr != 0 {
		return fmt.Errorf("SetCurrentProcessExplicitAppUserModelID failed: HRESULT 0x%X", uint32(hr))
	}
	return nil
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

func cleanupBalloonIconLocked(hwnd uintptr) {
	del := notifyIconData{
		cbSize:   uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:     hwnd,
		uID:      1,
		uFlags:   nifGUID,
		guidItem: pauseNotifyGUID,
	}
	shellNotifyIcon(nimDelete, &del)
	_, _, _ = procDestroyWindow.Call(hwnd)
}
