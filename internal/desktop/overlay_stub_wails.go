//go:build wails && !darwin && !windows && !linux

package desktop

type noopBreakOverlayController struct{}

func NewBreakOverlayController() BreakOverlayController {
	return noopBreakOverlayController{}
}

func (noopBreakOverlayController) Init(func()) {}
func (noopBreakOverlayController) Show(bool, string, string, string) bool {
	return false
}
func (noopBreakOverlayController) Hide()          {}
func (noopBreakOverlayController) Destroy()       {}
func (noopBreakOverlayController) IsNative() bool { return false }
