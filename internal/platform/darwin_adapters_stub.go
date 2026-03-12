//go:build !darwin

package platform

func newDarwinAdapters(_ string) Adapters {
	return Adapters{
		IdleProvider:   NoopIdleProvider{},
		Notifier:       NoopNotifier{},
		SoundPlayer:    NoopSoundPlayer{},
		StartupManager: NoopStartupManager{},
	}
}
