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
	return settingsdomain.Settings{GlobalEnabled: true}
}

func (s *settingsRepoStub) UpdateSettings(_ context.Context, patch settingsdomain.SettingsPatch) (settingsdomain.Settings, error) {
	s.gotPatch = patch
	return settingsdomain.Settings{GlobalEnabled: patch.GlobalEnabled != nil && *patch.GlobalEnabled}, nil
}

func (s *settingsRepoStub) SyncPlatformSettings(_ context.Context) error { return nil }

func (s *settingsRepoStub) GetLaunchAtLogin(_ context.Context) (bool, error) { return true, nil }

func (s *settingsRepoStub) SetLaunchAtLogin(_ context.Context, enabled bool) (bool, error) {
	return enabled, nil
}

func TestServiceUpdateForwardsPatch(t *testing.T) {
	repo := &settingsRepoStub{}
	svc, err := NewService(repo)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	enabled := false
	patch := settingsdomain.SettingsPatch{GlobalEnabled: &enabled}

	updated, err := svc.Update(context.Background(), patch)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if repo.gotPatch.GlobalEnabled == nil || *repo.gotPatch.GlobalEnabled != enabled {
		t.Fatalf("expected patch globalEnabled=%t to be forwarded", enabled)
	}
	if updated.GlobalEnabled != enabled {
		t.Fatalf("expected updated globalEnabled=%t, got %t", enabled, updated.GlobalEnabled)
	}
}
