package ports

import (
	"context"

	settingsdomain "pause/internal/backend/domain/settings"
)

type SettingsRepository interface {
	GetSettings(ctx context.Context) settingsdomain.Settings
	UpdateSettings(ctx context.Context, patch settingsdomain.SettingsPatch) (settingsdomain.Settings, error)
	SyncPlatformSettings(ctx context.Context) error
	GetLaunchAtLogin(ctx context.Context) (bool, error)
	SetLaunchAtLogin(ctx context.Context, enabled bool) (bool, error)
}
