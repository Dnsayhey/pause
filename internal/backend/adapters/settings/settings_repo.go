package settingsadapter

import (
	"context"
	"errors"

	settingsdomain "pause/internal/backend/domain/settings"
	"pause/internal/backend/ports"
)

var (
	errSettingsStoreUnavailable = errors.New("settings store unavailable")
)

type SettingsStore interface {
	WasCreated() bool
	Get() settingsdomain.Settings
	Update(patch settingsdomain.SettingsPatch) (settingsdomain.Settings, error)
}

type noopStartupManager struct{}

func (noopStartupManager) SetLaunchAtLogin(bool) error { return nil }
func (noopStartupManager) GetLaunchAtLogin() (bool, error) {
	return false, nil
}

type SettingsRepository struct {
	store          SettingsStore
	startupManager ports.StartupManager
}

var _ ports.SettingsRepository = (*SettingsRepository)(nil)

func NewSettingsRepository(store SettingsStore, startupManager ports.StartupManager) *SettingsRepository {
	if startupManager == nil {
		startupManager = noopStartupManager{}
	}
	return &SettingsRepository{
		store:          store,
		startupManager: startupManager,
	}
}

func (r *SettingsRepository) GetSettings(ctx context.Context) settingsdomain.Settings {
	_ = ctx
	if err := r.ensureStore(); err != nil {
		return settingsdomain.Settings{}
	}
	return r.store.Get()
}

func (r *SettingsRepository) UpdateSettings(ctx context.Context, patch settingsdomain.SettingsPatch) (settingsdomain.Settings, error) {
	_ = ctx
	if err := r.ensureStore(); err != nil {
		return settingsdomain.Settings{}, err
	}
	return r.store.Update(patch)
}

func (r *SettingsRepository) SyncPlatformSettings(ctx context.Context) error {
	_ = ctx
	if err := r.ensureStore(); err != nil {
		return err
	}

	if !r.store.WasCreated() {
		return nil
	}
	return r.startupManager.SetLaunchAtLogin(true)
}

func (r *SettingsRepository) GetLaunchAtLogin(ctx context.Context) (bool, error) {
	_ = ctx
	if err := r.ensureStore(); err != nil {
		return false, err
	}
	return r.startupManager.GetLaunchAtLogin()
}

func (r *SettingsRepository) SetLaunchAtLogin(ctx context.Context, enabled bool) (bool, error) {
	_ = ctx
	if err := r.ensureStore(); err != nil {
		return false, err
	}
	if err := r.startupManager.SetLaunchAtLogin(enabled); err != nil {
		return false, err
	}
	return r.startupManager.GetLaunchAtLogin()
}

func (r *SettingsRepository) ensureStore() error {
	if r == nil || r.store == nil {
		return errSettingsStoreUnavailable
	}
	return nil
}
