//go:build darwin && wails

package app

import "pause/internal/core/config"

func decorateRuntimeStateForPlatform(state config.RuntimeState) config.RuntimeState {
	state.OverlayNative = true
	return state
}
