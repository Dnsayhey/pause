package api

import (
	"context"
	"errors"
	"strings"

	settingsdomain "pause/internal/backend/domain/settings"
	"pause/internal/backend/ports"
)

var ErrNotificationSettingsUnavailable = errors.New("notification settings are unavailable on this platform")

type Adapters struct {
	IdleProvider                   ports.IdleProvider
	LockStateProvider              ports.LockStateProvider
	Notifier                       ports.Notifier
	NotificationCapabilityProvider ports.NotificationCapabilityProvider
	SoundPlayer                    ports.SoundPlayer
	StartupManager                 ports.StartupManager
}

func (a Adapters) WithDefaults() Adapters {
	if a.IdleProvider == nil {
		a.IdleProvider = NoopIdleProvider{}
	}
	if a.LockStateProvider == nil {
		a.LockStateProvider = NoopLockStateProvider{}
	}
	if a.Notifier == nil {
		a.Notifier = NoopNotifier{}
	}
	if a.NotificationCapabilityProvider == nil {
		a.NotificationCapabilityProvider = NoopNotificationCapabilityProvider{}
	}
	if a.SoundPlayer == nil {
		a.SoundPlayer = NoopSoundPlayer{}
	}
	if a.StartupManager == nil {
		a.StartupManager = NoopStartupManager{}
	}
	return a
}

type NoopIdleProvider struct{}

func (NoopIdleProvider) CurrentIdleSeconds() int { return 0 }

type NoopLockStateProvider struct{}

func (NoopLockStateProvider) IsScreenLocked() bool { return false }

type NoopNotifier struct{}

func (NoopNotifier) ShowReminder(context.Context, string, string) error { return nil }

type NoopNotificationCapabilityProvider struct{}

func UnsupportedNotificationCapability(reason string) ports.NotificationCapability {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "notification capability unavailable on this platform"
	}
	return ports.NotificationCapability{
		PermissionState: ports.NotificationPermissionUnknown,
		CanRequest:      false,
		CanOpenSettings: false,
		Reason:          reason,
	}
}

func (NoopNotificationCapabilityProvider) GetNotificationCapability() ports.NotificationCapability {
	return UnsupportedNotificationCapability("")
}

func (NoopNotificationCapabilityProvider) RequestNotificationPermission() (ports.NotificationCapability, error) {
	return UnsupportedNotificationCapability(""), nil
}

func (NoopNotificationCapabilityProvider) OpenNotificationSettings() error {
	return ErrNotificationSettingsUnavailable
}

type NoopSoundPlayer struct{}

func (NoopSoundPlayer) PlayBreakEnd(_ settingsdomain.SoundSettings) error { return nil }

type NoopStartupManager struct{}

func (NoopStartupManager) SetLaunchAtLogin(_ bool) error { return nil }
func (NoopStartupManager) GetLaunchAtLogin() (bool, error) {
	return false, nil
}
