package settings

import (
	"context"
	"testing"

	settingsdomain "pause/internal/backend/domain/settings"
)

type settingsRepoStub struct {
	gotPatch settingsdomain.SettingsPatch
}

func (s *settingsRepoStub) GetSettings(_ context.Context) settingsdomain.Settings {
	return settingsdomain.DefaultSettings()
}

func (s *settingsRepoStub) UpdateSettings(_ context.Context, patch settingsdomain.SettingsPatch) (settingsdomain.Settings, error) {
	s.gotPatch = patch
	return settingsdomain.DefaultSettings().ApplyPatch(patch), nil
}

type platformSyncerStub struct{}

func (platformSyncerStub) SyncPlatformSettings(_ context.Context) error { return nil }

type startupManagerStub struct {
	current bool
}

func (s *startupManagerStub) GetLaunchAtLogin() (bool, error) { return s.current, nil }
func (s *startupManagerStub) SetLaunchAtLogin(enabled bool) error {
	s.current = enabled
	return nil
}

func TestSettingsService_UpdateForwardsPatch(t *testing.T) {
	repo := &settingsRepoStub{}
	svc, err := NewService(repo, platformSyncerStub{}, &startupManagerStub{})
	if err != nil {
		t.Fatalf("NewService() err=%v", err)
	}

	enabled := false
	updated, err := svc.Update(context.Background(), settingsdomain.SettingsPatch{
		Sound: &settingsdomain.SoundSettingsPatch{Enabled: &enabled},
	})
	if err != nil {
		t.Fatalf("Update() err=%v", err)
	}
	if repo.gotPatch.Sound == nil || repo.gotPatch.Sound.Enabled == nil || *repo.gotPatch.Sound.Enabled != enabled {
		t.Fatalf("forwarded patch mismatch")
	}
	if updated.Sound.Enabled != enabled {
		t.Fatalf("updated value mismatch: got=%t want=%t", updated.Sound.Enabled, enabled)
	}
}

func TestSettingsService_LaunchAtLoginUsesStartupManager(t *testing.T) {
	startup := &startupManagerStub{}
	svc, err := NewService(&settingsRepoStub{}, platformSyncerStub{}, startup)
	if err != nil {
		t.Fatalf("NewService() err=%v", err)
	}

	actual, err := svc.SetLaunchAtLogin(context.Background(), true)
	if err != nil {
		t.Fatalf("SetLaunchAtLogin() err=%v", err)
	}
	if !actual {
		t.Fatalf("expected launch-at-login to be true")
	}
}
