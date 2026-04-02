//go:build linux

package platform

import (
	"pause/internal/platform/api"
	"pause/internal/platform/linux"
)

func NewAdapters(appID string) api.Adapters {
	return withNotificationCapabilityOverride(linux.NewAdapters(appID))
}
