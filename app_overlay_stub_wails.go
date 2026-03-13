//go:build wails && !darwin

package main

type noopBreakOverlayController struct{}

func newBreakOverlayController() breakOverlayController {
	return noopBreakOverlayController{}
}

func (noopBreakOverlayController) Init(func()) {}
func (noopBreakOverlayController) Show(bool, string, string, string) bool {
	return false
}
func (noopBreakOverlayController) Hide()          {}
func (noopBreakOverlayController) Destroy()       {}
func (noopBreakOverlayController) IsNative() bool { return false }
