package settings

import (
	"context"
	"errors"

	settingsdomain "pause/internal/backend/domain/settings"
	"pause/internal/backend/ports"
	internalctx "pause/internal/backend/usecase/internalctx"
)

type Service struct {
	storeRepo      ports.SettingsStoreRepository
	platformSyncer ports.PlatformSettingsSyncer
	startupManager ports.StartupManager
}

func NewService(storeRepo ports.SettingsStoreRepository, platformSyncer ports.PlatformSettingsSyncer, startupManager ports.StartupManager) (*Service, error) {
	if storeRepo == nil {
		return nil, errors.New("settings store repository is required")
	}
	if platformSyncer == nil {
		return nil, errors.New("platform settings syncer is required")
	}
	if startupManager == nil {
		return nil, errors.New("startup manager is required")
	}
	return &Service{
		storeRepo:      storeRepo,
		platformSyncer: platformSyncer,
		startupManager: startupManager,
	}, nil
}

func (s *Service) Get(ctx context.Context) settingsdomain.Settings {
	return s.storeRepo.GetSettings(internalctx.OrBackground(ctx))
}

func (s *Service) Update(ctx context.Context, patch settingsdomain.SettingsPatch) (settingsdomain.Settings, error) {
	return s.storeRepo.UpdateSettings(internalctx.OrBackground(ctx), patch)
}

func (s *Service) SyncPlatformSettings(ctx context.Context) error {
	return s.platformSyncer.SyncPlatformSettings(internalctx.OrBackground(ctx))
}

func (s *Service) GetLaunchAtLogin(ctx context.Context) (bool, error) {
	return s.startupManager.GetLaunchAtLogin()
}

func (s *Service) SetLaunchAtLogin(ctx context.Context, enabled bool) (bool, error) {
	_ = internalctx.OrBackground(ctx)
	if err := s.startupManager.SetLaunchAtLogin(enabled); err != nil {
		return false, err
	}
	return s.startupManager.GetLaunchAtLogin()
}
