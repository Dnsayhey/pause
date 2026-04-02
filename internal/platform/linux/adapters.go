//go:build linux

package linux

import (
	"pause/internal/platform/api"
)

func NewAdapters(_ string) api.Adapters {
	return api.Adapters{}
}
