//go:build windows && wails

package app

import "pause/internal/core/state"

func decorateRuntimeStateForPlatform(state state.RuntimeState) state.RuntimeState {
	state.OverlayNative = true
	return state
}
