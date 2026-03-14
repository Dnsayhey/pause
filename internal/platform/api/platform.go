package api

import "pause/internal/core/config"

type Adapters struct {
	IdleProvider   IdleProvider
	Notifier       Notifier
	SoundPlayer    SoundPlayer
	StartupManager StartupManager
}

type IdleProvider interface {
	CurrentIdleSeconds() int
}

type Notifier interface {
	ShowReminder(title, body string) error
}

type SoundPlayer interface {
	PlayBreakEnd(sound config.SoundSettings) error
}

type StartupManager interface {
	SetLaunchAtLogin(enabled bool) error
	GetLaunchAtLogin() (bool, error)
}

type NoopIdleProvider struct{}

func (NoopIdleProvider) CurrentIdleSeconds() int { return 0 }

type NoopNotifier struct{}

func (NoopNotifier) ShowReminder(_, _ string) error { return nil }

type NoopSoundPlayer struct{}

func (NoopSoundPlayer) PlayBreakEnd(_ config.SoundSettings) error { return nil }

type NoopStartupManager struct{}

func (NoopStartupManager) SetLaunchAtLogin(_ bool) error { return nil }
func (NoopStartupManager) GetLaunchAtLogin() (bool, error) {
	return false, nil
}
