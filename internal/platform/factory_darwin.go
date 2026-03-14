//go:build darwin

package platform

import "pause/internal/platform/darwin"

func NewAdapters(appID string) Adapters {
	return darwin.NewAdapters(appID)
}
