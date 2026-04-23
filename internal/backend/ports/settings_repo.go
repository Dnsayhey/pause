package ports

import (
	"context"

	settingsdomain "pause/internal/backend/domain/settings"
)

type SettingsStoreRepository interface {
	GetSettings(ctx context.Context) settingsdomain.Settings
	UpdateSettings(ctx context.Context, patch settingsdomain.SettingsPatch) (settingsdomain.Settings, error)
}

type PlatformSettingsSyncer interface {
	SyncPlatformSettings(ctx context.Context) error
}
