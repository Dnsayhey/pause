//go:build linux

package platform

import "pause/internal/platform/linux"

func NewAdapters(appID string) Adapters {
	return linux.NewAdapters(appID)
}
