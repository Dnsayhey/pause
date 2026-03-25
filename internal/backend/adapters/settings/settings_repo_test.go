package settingsadapter

import (
	"context"
	"errors"
	"testing"

	settingsdomain "pause/internal/backend/domain/settings"
)

type fakeSettingsStore struct {
	created bool
	current settingsdomain.Settings
}

func (s *fakeSettingsStore) WasCreated() bool { return s.created }
func (s *fakeSettingsStore) Get() settingsdomain.Settings {
	return s.current
}
func (s *fakeSettingsStore) Update(patch settingsdomain.SettingsPatch) (settingsdomain.Settings, error) {
	s.current = s.current.ApplyPatch(patch)
	return s.current, nil
}

type fakeStartupManager struct {
	setCalls int
	lastSet  bool
	getCalls int
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

func TestSyncPlatformSettingsOnlyOnFirstRun(t *testing.T) {
	manager := &fakeStartupManager{}
	firstStore := &fakeSettingsStore{created: true, current: settingsdomain.DefaultSettings()}
	firstRepo := NewSettingsRepository(firstStore, manager)

	if err := firstRepo.SyncPlatformSettings(context.Background()); err != nil {
		t.Fatalf("SyncPlatformSettings(first run) error = %v", err)
	}
	if manager.setCalls != 1 || !manager.lastSet {
		t.Fatalf("expected first run to set launch at login=true once")
	}

	manager.setCalls = 0
	existingStore := &fakeSettingsStore{created: false, current: settingsdomain.DefaultSettings()}
	existingRepo := NewSettingsRepository(existingStore, manager)
	if err := existingRepo.SyncPlatformSettings(context.Background()); err != nil {
		t.Fatalf("SyncPlatformSettings(existing config) error = %v", err)
	}
	if manager.setCalls != 0 {
		t.Fatalf("expected existing config to skip launch-at-login sync")
	}
}

func TestSetLaunchAtLoginVerifiesCurrentState(t *testing.T) {
	manager := &fakeStartupManager{}
	repo := NewSettingsRepository(&fakeSettingsStore{current: settingsdomain.DefaultSettings()}, manager)

	actual, err := repo.SetLaunchAtLogin(context.Background(), true)
	if err != nil {
		t.Fatalf("SetLaunchAtLogin() error = %v", err)
	}
	if !actual {
		t.Fatalf("expected launch-at-login state to be true")
	}
	if manager.setCalls != 1 || manager.getCalls != 1 {
		t.Fatalf("expected set/get calls to both be 1, got set=%d get=%d", manager.setCalls, manager.getCalls)
	}
}

func TestSetLaunchAtLoginReturnsSetError(t *testing.T) {
	expected := errors.New("set failed")
	manager := &fakeStartupManager{setErr: expected}
	repo := NewSettingsRepository(&fakeSettingsStore{current: settingsdomain.DefaultSettings()}, manager)

	_, err := repo.SetLaunchAtLogin(context.Background(), true)
	if !errors.Is(err, expected) {
		t.Fatalf("expected set error %v, got %v", expected, err)
	}
}
