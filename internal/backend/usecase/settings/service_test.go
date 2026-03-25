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

func (s *settingsRepoStub) SyncPlatformSettings(_ context.Context) error     { return nil }
func (s *settingsRepoStub) GetLaunchAtLogin(_ context.Context) (bool, error) { return true, nil }
func (s *settingsRepoStub) SetLaunchAtLogin(_ context.Context, enabled bool) (bool, error) {
	return enabled, nil
}

func TestSettingsService_UpdateForwardsPatch(t *testing.T) {
	repo := &settingsRepoStub{}
	svc, err := NewService(repo)
	if err != nil {
		t.Fatalf("NewService() err=%v", err)
	}

	enabled := false
	updated, err := svc.Update(context.Background(), settingsdomain.SettingsPatch{GlobalEnabled: &enabled})
	if err != nil {
		t.Fatalf("Update() err=%v", err)
	}
	if repo.gotPatch.GlobalEnabled == nil || *repo.gotPatch.GlobalEnabled != enabled {
		t.Fatalf("forwarded patch mismatch")
	}
	if updated.GlobalEnabled != enabled {
		t.Fatalf("updated value mismatch: got=%t want=%t", updated.GlobalEnabled, enabled)
	}
}
