//go:build !darwin || !wails

package main

import "pause/internal/config"

func decorateRuntimeStateForPlatform(state config.RuntimeState) config.RuntimeState {
	return state
}
