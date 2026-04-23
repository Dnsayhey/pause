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
	store SettingsStore
}

var _ ports.SettingsStoreRepository = (*SettingsRepository)(nil)

func NewSettingsRepository(store SettingsStore) *SettingsRepository {
	return &SettingsRepository{
		store: store,
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

type PlatformSettingsSyncer struct {
	store          SettingsStore
	startupManager ports.StartupManager
}

var _ ports.PlatformSettingsSyncer = (*PlatformSettingsSyncer)(nil)

func NewPlatformSettingsSyncer(store SettingsStore, startupManager ports.StartupManager) *PlatformSettingsSyncer {
	if startupManager == nil {
		startupManager = noopStartupManager{}
	}
	return &PlatformSettingsSyncer{
		store:          store,
		startupManager: startupManager,
	}
}

func (r *PlatformSettingsSyncer) SyncPlatformSettings(ctx context.Context) error {
	_ = ctx
	if err := r.ensureStore(); err != nil {
		return err
	}

	if !r.store.WasCreated() {
		return nil
	}
	return r.startupManager.SetLaunchAtLogin(true)
}

func (r *SettingsRepository) ensureStore() error {
	if r == nil || r.store == nil {
		return errSettingsStoreUnavailable
	}
	return nil
}

func (r *PlatformSettingsSyncer) ensureStore() error {
	if r == nil || r.store == nil {
		return errSettingsStoreUnavailable
	}
	return nil
}
