//go:build darwin && wails

package app

func decorateRuntimeStateForPlatform(state RuntimeState) RuntimeState {
	state.OverlayNative = true
	return state
}
