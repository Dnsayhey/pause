//go:build darwin

package darwin

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"pause/internal/backend/domain/settings"
	"pause/internal/backend/ports"
	"pause/internal/meta"
	"pause/internal/platform/api"
)

const (
	defaultMacSound = "/System/Library/Sounds/Glass.aiff"
	idleSampleTTL   = 2 * time.Second
)

type darwinIdleProvider struct {
	mu           sync.Mutex
	lastSampleAt time.Time
	lastIdleSec  int
}

type darwinNotifier struct{}
type darwinNotificationCapabilityProvider struct {
	appID string
}

type darwinSoundPlayer struct{}

type darwinStartupManager struct {
	appID          string
	helperBundleID string
}

var errSMUnsupported = errors.New("SMAppService is unavailable")

func NewAdapters(appID string) api.Adapters {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		appID = meta.EffectiveAppBundleID()
	}
	helperBundleID := appID + ".loginhelper"
	return api.Adapters{
		IdleProvider:                   &darwinIdleProvider{},
		LockStateProvider:              darwinLockStateProvider{},
		Notifier:                       darwinNotifier{},
		NotificationCapabilityProvider: darwinNotificationCapabilityProvider{appID: appID},
		SoundPlayer:                    darwinSoundPlayer{},
		StartupManager:                 darwinStartupManager{appID: appID, helperBundleID: helperBundleID},
	}
}

func (p *darwinIdleProvider) CurrentIdleSeconds() int {
	now := time.Now()

	p.mu.Lock()
	if !p.lastSampleAt.IsZero() {
		age := now.Sub(p.lastSampleAt)
		if age <= idleSampleTTL {
			idleSec := p.lastIdleSec + int(age.Seconds())
			p.mu.Unlock()
			return idleSec
		}
	}
	p.mu.Unlock()

	idleSec, ok := queryDarwinIdleSeconds()
	if !ok {
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.lastSampleAt.IsZero() {
			return 0
		}
		age := now.Sub(p.lastSampleAt)
		return p.lastIdleSec + int(age.Seconds())
	}

	p.mu.Lock()
	p.lastSampleAt = now
	p.lastIdleSec = idleSec
	p.mu.Unlock()
	return idleSec
}

func queryDarwinIdleSeconds() (int, bool) {
	idleNs, ok := queryDarwinIdleNanoseconds()
	if !ok {
		return 0, false
	}
	return idleSecondsFromNanoseconds(idleNs), true
}

func idleSecondsFromNanoseconds(ns uint64) int {
	if ns == 0 {
		return 0
	}
	return int(ns / 1_000_000_000)
}

func (darwinNotifier) ShowReminder(title, body string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Pause"
	}
	body = strings.TrimSpace(body)
	if body == "" {
		body = "Break started"
	}
	return showDarwinUserNotification(title, body)
}

func (p darwinNotificationCapabilityProvider) GetNotificationCapability() ports.NotificationCapability {
	status, err := darwinNotificationAuthorizationStatus()
	if err != nil {
		return ports.NotificationCapability{
			PermissionState: ports.NotificationPermissionUnknown,
			CanRequest:      false,
			CanOpenSettings: true,
			Reason:          err.Error(),
		}
	}
	return darwinCapabilityFromAuthorizationStatus(status)
}

func (p darwinNotificationCapabilityProvider) RequestNotificationPermission() (ports.NotificationCapability, error) {
	current := p.GetNotificationCapability()
	if current.PermissionState != ports.NotificationPermissionNotDetermined || !current.CanRequest {
		return current, nil
	}
	granted, err := darwinRequestNotificationAuthorization()
	if err != nil {
		return current, err
	}
	next := p.GetNotificationCapability()
	if granted {
		next.PermissionState = ports.NotificationPermissionAuthorized
		next.CanRequest = false
	}
	return next, nil
}

func (p darwinNotificationCapabilityProvider) OpenNotificationSettings() error {
	return darwinOpenNotificationSettings(p.appID)
}

func darwinCapabilityFromAuthorizationStatus(status int) ports.NotificationCapability {
	switch status {
	case darwinNotificationStatusNotDetermined:
		return ports.NotificationCapability{
			PermissionState: ports.NotificationPermissionNotDetermined,
			CanRequest:      true,
			CanOpenSettings: true,
		}
	case darwinNotificationStatusDenied:
		return ports.NotificationCapability{
			PermissionState: ports.NotificationPermissionDenied,
			CanRequest:      false,
			CanOpenSettings: true,
			Reason:          "notification permission denied",
		}
	case darwinNotificationStatusAuthorized, darwinNotificationStatusProvisional, darwinNotificationStatusEphemeral:
		return ports.NotificationCapability{
			PermissionState: ports.NotificationPermissionAuthorized,
			CanRequest:      false,
			CanOpenSettings: true,
		}
	default:
		return ports.NotificationCapability{
			PermissionState: ports.NotificationPermissionUnknown,
			CanRequest:      false,
			CanOpenSettings: true,
			Reason:          fmt.Sprintf("unknown notification authorization status: %d", status),
		}
	}
}

func (darwinSoundPlayer) PlayBreakEnd(sound settings.SoundSettings) error {
	if !sound.Enabled {
		return nil
	}
	volume := float64(sound.Volume) / 100.0
	if volume < 0 {
		volume = 0
	}
	if volume > 1 {
		volume = 1
	}
	return exec.Command("afplay", "-v", fmt.Sprintf("%.2f", volume), defaultMacSound).Start()
}

func (s darwinStartupManager) SetLaunchAtLogin(enabled bool) error {
	if enabled {
		execPath, err := os.Executable()
		if err != nil {
			return err
		}
		resolvedExecPath, err := filepath.EvalSymlinks(execPath)
		if err == nil && strings.TrimSpace(resolvedExecPath) != "" {
			execPath = resolvedExecPath
		}
		if err := validateStartupExecutablePath(execPath); err != nil {
			return err
		}
	}

	if err := smSetLaunchAtLogin(s.helperBundleID, enabled); err != nil {
		if enabled {
			return wrapLaunchAtLoginEnableError(err)
		}
		return err
	}

	actual, err := smGetLaunchAtLogin(s.helperBundleID)
	if err != nil {
		return err
	}
	if actual != enabled {
		if enabled {
			return wrapLaunchAtLoginEnableError(nil)
		}
		return errors.New("launch-at-login remained enabled after disabling")
	}
	return nil
}

func (s darwinStartupManager) GetLaunchAtLogin() (bool, error) {
	return smGetLaunchAtLogin(s.helperBundleID)
}

func wrapLaunchAtLoginEnableError(cause error) error {
	const msg = "macOS blocked enabling launch at login. Please enable Pause in System Settings > General > Login Items > Allow in the Background"
	if cause == nil {
		return errors.New(msg)
	}
	return fmt.Errorf("%s: %v", msg, cause)
}

func validateStartupExecutablePath(execPath string) error {
	if strings.HasPrefix(execPath, "/Volumes/") {
		return errors.New("Pause is running from a mounted volume; move Pause.app to /Applications and re-enable launch at login")
	}
	if strings.Contains(execPath, "/AppTranslocation/") {
		return errors.New("Pause is running from App Translocation; move Pause.app to /Applications and re-enable launch at login")
	}
	return nil
}
