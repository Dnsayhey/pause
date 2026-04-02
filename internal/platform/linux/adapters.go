//go:build linux

package linux

import (
	"pause/internal/platform/api"
	"pause/internal/platform/fallbacks"
)

func NewAdapters(_ string) api.Adapters {
	return api.Adapters{
		IdleProvider:                   api.NoopIdleProvider{},
		LockStateProvider:              api.NoopLockStateProvider{},
		Notifier:                       api.NoopNotifier{},
		NotificationCapabilityProvider: fallbacks.NoopNotificationCapabilityProvider{},
		SoundPlayer:                    api.NoopSoundPlayer{},
		StartupManager:                 api.NoopStartupManager{},
	}
}
