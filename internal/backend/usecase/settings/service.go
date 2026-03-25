package settings

import (
	"context"
	"errors"

	settingsdomain "pause/internal/backend/domain/settings"
	"pause/internal/backend/ports"
)

type Service struct {
	repo ports.SettingsRepository
}

func NewService(repo ports.SettingsRepository) (*Service, error) {
	if repo == nil {
		return nil, errors.New("settings repository is required")
	}
	return &Service{repo: repo}, nil
}

func (s *Service) Get(ctx context.Context) settingsdomain.Settings {
	return s.repo.GetSettings(normalizeContext(ctx))
}

func (s *Service) Update(ctx context.Context, patch settingsdomain.SettingsPatch) (settingsdomain.Settings, error) {
	return s.repo.UpdateSettings(normalizeContext(ctx), patch)
}

func (s *Service) SyncPlatformSettings(ctx context.Context) error {
	return s.repo.SyncPlatformSettings(normalizeContext(ctx))
}

func (s *Service) GetLaunchAtLogin(ctx context.Context) (bool, error) {
	return s.repo.GetLaunchAtLogin(normalizeContext(ctx))
}

func (s *Service) SetLaunchAtLogin(ctx context.Context, enabled bool) (bool, error) {
	return s.repo.SetLaunchAtLogin(normalizeContext(ctx), enabled)
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}
