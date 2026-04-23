package settingsadapter

import (
	"context"
	"testing"

	settingsdomain "pause/internal/backend/domain/settings"
)

type fakeSettingsStore struct {
	created bool
	current settingsdomain.Settings
}

func (s *fakeSettingsStore) WasCreated() bool             { return s.created }
func (s *fakeSettingsStore) Get() settingsdomain.Settings { return s.current }
func (s *fakeSettingsStore) Update(patch settingsdomain.SettingsPatch) (settingsdomain.Settings, error) {
	s.current = s.current.ApplyPatch(patch)
	return s.current, nil
}

type fakeStartupManager struct {
	setCalls int
	getCalls int
	lastSet  bool
	current  bool
	setErr   error
	getErr   error
}

func (m *fakeStartupManager) SetLaunchAtLogin(enabled bool) error {
	m.setCalls++
	m.lastSet = enabled
	if m.setErr != nil {
		return m.setErr
	}
	m.current = enabled
	return nil
}

func (m *fakeStartupManager) GetLaunchAtLogin() (bool, error) {
	m.getCalls++
	if m.getErr != nil {
		return false, m.getErr
	}
	return m.current, nil
}

func TestSettingsRepository_SyncFirstInstallOnly(t *testing.T) {
	manager := &fakeStartupManager{}
	first := NewPlatformSettingsSyncer(&fakeSettingsStore{created: true, current: settingsdomain.DefaultSettings()}, manager)
	if err := first.SyncPlatformSettings(context.Background()); err != nil {
		t.Fatalf("SyncPlatformSettings(first) err=%v", err)
	}
	if manager.setCalls != 1 || !manager.lastSet {
		t.Fatalf("first install sync mismatch")
	}

	manager.setCalls = 0
	existing := NewPlatformSettingsSyncer(&fakeSettingsStore{created: false, current: settingsdomain.DefaultSettings()}, manager)
	if err := existing.SyncPlatformSettings(context.Background()); err != nil {
		t.Fatalf("SyncPlatformSettings(existing) err=%v", err)
	}
	if manager.setCalls != 0 {
		t.Fatalf("existing config should not trigger startup sync")
	}
}

func TestSettingsRepository_UpdateSettings(t *testing.T) {
	repo := NewSettingsRepository(&fakeSettingsStore{current: settingsdomain.DefaultSettings()})

	enabled := false
	got, err := repo.UpdateSettings(context.Background(), settingsdomain.SettingsPatch{
		Sound: &settingsdomain.SoundSettingsPatch{Enabled: &enabled},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() err=%v", err)
	}
	if got.Sound.Enabled {
		t.Fatalf("expected updated sound setting to be false")
	}
}
