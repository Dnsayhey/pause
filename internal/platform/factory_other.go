//go:build !darwin && !windows && !linux

package platform

import "pause/internal/platform/api"

func NewAdapters(_ string) api.Adapters {
	return withNotificationCapabilityOverride(api.Adapters{}).WithDefaults()
}
