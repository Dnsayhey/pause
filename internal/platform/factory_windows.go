//go:build windows

package platform

import (
	"pause/internal/platform/api"
	"pause/internal/platform/windows"
)

func NewAdapters(appID string) api.Adapters {
	return withNotificationCapabilityOverride(windows.NewAdapters(appID))
}
