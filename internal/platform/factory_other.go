//go:build !darwin && !windows && !linux

package platform

import "pause/internal/platform/api"

func NewAdapters(_ string) Adapters {
	return api.Adapters{
		IdleProvider:   api.NoopIdleProvider{},
		Notifier:       api.NoopNotifier{},
		SoundPlayer:    api.NoopSoundPlayer{},
		StartupManager: api.NoopStartupManager{},
	}
}
