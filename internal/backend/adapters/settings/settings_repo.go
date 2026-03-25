package settingsadapter

import (
	"context"
	"errors"

	settingsdomain "pause/internal/backend/domain/settings"
	"pause/internal/backend/ports"
	runtimeengine "pause/internal/backend/runtime/engine"
)

var errEngineUnavailable = errors.New("engine unavailable")

type SettingsRepository struct {
	engine *runtimeengine.Engine
}

var _ ports.SettingsRepository = (*SettingsRepository)(nil)

func NewSettingsRepository(engine *runtimeengine.Engine) *SettingsRepository {
	return &SettingsRepository{engine: engine}
}

func (r *SettingsRepository) GetSettings(ctx context.Context) settingsdomain.Settings {
	_ = ctx
	if err := r.ensureEngine(); err != nil {
		return settingsdomain.Settings{}
	}
	return r.engine.GetSettings()
}

func (r *SettingsRepository) UpdateSettings(ctx context.Context, patch settingsdomain.SettingsPatch) (settingsdomain.Settings, error) {
	_ = ctx
	if err := r.ensureEngine(); err != nil {
		return settingsdomain.Settings{}, err
	}
	return r.engine.UpdateSettings(patch)
}

func (r *SettingsRepository) SyncPlatformSettings(ctx context.Context) error {
	_ = ctx
	if err := r.ensureEngine(); err != nil {
		return err
	}
	return r.engine.SyncPlatformSettings()
}

func (r *SettingsRepository) GetLaunchAtLogin(ctx context.Context) (bool, error) {
	_ = ctx
	if err := r.ensureEngine(); err != nil {
		return false, err
	}
	return r.engine.GetLaunchAtLogin()
}

func (r *SettingsRepository) SetLaunchAtLogin(ctx context.Context, enabled bool) (bool, error) {
	_ = ctx
	if err := r.ensureEngine(); err != nil {
		return false, err
	}
	return r.engine.SetLaunchAtLogin(enabled)
}

func (r *SettingsRepository) ensureEngine() error {
	if r == nil || r.engine == nil {
		return errEngineUnavailable
	}
	return nil
}
