package api

import "testing"

func TestAdaptersWithDefaults_FillsMissingProviders(t *testing.T) {
	adapters := (Adapters{}).WithDefaults()

	if adapters.IdleProvider == nil {
		t.Fatalf("IdleProvider should be set")
	}
	if adapters.LockStateProvider == nil {
		t.Fatalf("LockStateProvider should be set")
	}
	if adapters.Notifier == nil {
		t.Fatalf("Notifier should be set")
	}
	if adapters.NotificationCapabilityProvider == nil {
		t.Fatalf("NotificationCapabilityProvider should be set")
	}
	if adapters.SoundPlayer == nil {
		t.Fatalf("SoundPlayer should be set")
	}
	if adapters.StartupManager == nil {
		t.Fatalf("StartupManager should be set")
	}
}
