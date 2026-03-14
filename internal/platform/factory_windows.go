//go:build windows

package platform

import "pause/internal/platform/windows"

func NewAdapters(appID string) Adapters {
	return windows.NewAdapters(appID)
}
