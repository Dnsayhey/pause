//go:build windows

package windows

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"

	"pause/internal/backend/domain/settings"
	"pause/internal/backend/ports"
	"pause/internal/logx"
	"pause/internal/meta"
	"pause/internal/platform/api"
)

const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

var (
	user32DLL                                   = syscall.NewLazyDLL("user32.dll")
	kernel32DLL                                 = syscall.NewLazyDLL("kernel32.dll")
	shell32DLL                                  = syscall.NewLazyDLL("shell32.dll")
	procGetLastInputInfo                        = user32DLL.NewProc("GetLastInputInfo")
	procGetTickCount64                          = kernel32DLL.NewProc("GetTickCount64")
	procMessageBeep                             = user32DLL.NewProc("MessageBeep")
	procCreateWindowExW                         = user32DLL.NewProc("CreateWindowExW")
	procDestroyWindow                           = user32DLL.NewProc("DestroyWindow")
	procShellExecuteW                           = shell32DLL.NewProc("ShellExecuteW")
	procSetCurrentProcessExplicitAppUserModelID = shell32DLL.NewProc("SetCurrentProcessExplicitAppUserModelID")
	showToastReminder                           = showWinRTToastReminderNative
	queryToastSetting                           = queryWindowsToastSettingNative
	openNotificationSettings                    = openWindowsNotificationSettingsNative
)

const (
	mbIconAsterisk = 0x00000040
)

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

func NewAdapters(appID string) api.Adapters {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		appID = meta.EffectiveAppBundleID()
	}
	toastID := toastAppID(appID)
	if err := setCurrentProcessAppUserModelID(toastID); err != nil {
		logx.Warnf("windows.notification.aumid_set_failed app_id=%s err=%v", toastID, err)
	}
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
	return showToastReminder(appID, title, body)
}

func (n windowsNotifier) GetNotificationCapability() ports.NotificationCapability {
	appID := toastAppID(n.appID)
	setting, err := queryToastSetting(appID)
	if err != nil {
		logx.Warnf("windows.notification.capability_lookup_failed app_id=%s err=%v", appID, err)
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

func toastAppID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "com.pause.app"
	}
	raw = strings.ReplaceAll(raw, "/", ".")
	return raw
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
