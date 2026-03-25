//go:build windows && wails

package app

func decorateRuntimeStateForPlatform(state RuntimeState) RuntimeState {
	state.OverlayNative = true
	return state
}
