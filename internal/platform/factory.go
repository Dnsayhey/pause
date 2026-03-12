package platform

import "runtime"

func NewAdapters(appID string) Adapters {
	switch runtime.GOOS {
	case "darwin":
		return newDarwinAdapters(appID)
	default:
		return Adapters{
			IdleProvider:   NoopIdleProvider{},
			Notifier:       NoopNotifier{},
			SoundPlayer:    NoopSoundPlayer{},
			StartupManager: NoopStartupManager{},
		}
	}
}
