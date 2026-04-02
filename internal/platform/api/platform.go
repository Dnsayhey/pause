package api

import (
	settingsdomain "pause/internal/backend/domain/settings"
	"pause/internal/backend/ports"
)

type Adapters struct {
	IdleProvider                   ports.IdleProvider
	LockStateProvider              ports.LockStateProvider
	Notifier                       ports.Notifier
	NotificationCapabilityProvider ports.NotificationCapabilityProvider
	SoundPlayer                    ports.SoundPlayer
	StartupManager                 ports.StartupManager
}

type NoopIdleProvider struct{}

func (NoopIdleProvider) CurrentIdleSeconds() int { return 0 }

type NoopLockStateProvider struct{}

func (NoopLockStateProvider) IsScreenLocked() bool { return false }

type NoopNotifier struct{}

func (NoopNotifier) ShowReminder(_, _ string) error { return nil }

type NoopSoundPlayer struct{}

func (NoopSoundPlayer) PlayBreakEnd(_ settingsdomain.SoundSettings) error { return nil }

type NoopStartupManager struct{}

func (NoopStartupManager) SetLaunchAtLogin(_ bool) error { return nil }
func (NoopStartupManager) GetLaunchAtLogin() (bool, error) {
	return false, nil
}
