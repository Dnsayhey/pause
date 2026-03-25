//go:build windows && wails

package app

import "pause/internal/backend/runtime/state"

func decorateRuntimeStateForPlatform(state state.RuntimeState) state.RuntimeState {
	state.OverlayNative = true
	return state
}
