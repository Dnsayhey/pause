//go:build darwin

package platform

import (
	"pause/internal/platform/api"
	"pause/internal/platform/darwin"
)

func NewAdapters(appID string) api.Adapters {
	return withNotificationCapabilityOverride(darwin.NewAdapters(appID)).WithDefaults()
}
