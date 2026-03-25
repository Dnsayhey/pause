package api

import "pause/internal/backend/domain/settings"

type Adapters struct {
	IdleProvider      IdleProvider
	LockStateProvider LockStateProvider
	Notifier          Notifier
	SoundPlayer       SoundPlayer
	StartupManager    StartupManager
}

type IdleProvider interface {
	CurrentIdleSeconds() int
}

type LockStateProvider interface {
	IsScreenLocked() bool
}

type Notifier interface {
	ShowReminder(title, body string) error
}

type SoundPlayer interface {
	PlayBreakEnd(sound settings.SoundSettings) error
}

type StartupManager interface {
	SetLaunchAtLogin(enabled bool) error
	GetLaunchAtLogin() (bool, error)
}

type NoopIdleProvider struct{}

func (NoopIdleProvider) CurrentIdleSeconds() int { return 0 }

type NoopLockStateProvider struct{}

func (NoopLockStateProvider) IsScreenLocked() bool { return false }

type NoopNotifier struct{}

func (NoopNotifier) ShowReminder(_, _ string) error { return nil }

type NoopSoundPlayer struct{}

func (NoopSoundPlayer) PlayBreakEnd(_ settings.SoundSettings) error { return nil }

type NoopStartupManager struct{}

func (NoopStartupManager) SetLaunchAtLogin(_ bool) error { return nil }
func (NoopStartupManager) GetLaunchAtLogin() (bool, error) {
	return false, nil
}
