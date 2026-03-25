//go:build !wails || (!darwin && !windows)

package app

func decorateRuntimeStateForPlatform(state RuntimeState) RuntimeState {
	return state
}
