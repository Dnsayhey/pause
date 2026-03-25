//go:build !wails || (!darwin && !windows)

package app

import "pause/internal/core/state"

func decorateRuntimeStateForPlatform(state state.RuntimeState) state.RuntimeState {
	return state
}
