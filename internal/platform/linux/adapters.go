//go:build linux

package linux

import "pause/internal/platform/api"

func NewAdapters(_ string) api.Adapters {
	return api.Adapters{
		IdleProvider:                   api.NoopIdleProvider{},
		LockStateProvider:              api.NoopLockStateProvider{},
		Notifier:                       api.NoopNotifier{},
		NotificationCapabilityProvider: api.NoopNotificationCapabilityProvider{},
		SoundPlayer:                    api.NoopSoundPlayer{},
		StartupManager:                 api.NoopStartupManager{},
	}
}
