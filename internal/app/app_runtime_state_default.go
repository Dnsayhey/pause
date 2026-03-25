//go:build !wails || (!darwin && !windows)

package app

import "pause/internal/backend/runtime/state"

func decorateRuntimeStateForPlatform(state state.RuntimeState) state.RuntimeState {
	return state
}
