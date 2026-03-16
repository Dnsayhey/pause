//go:build wails && !darwin && !windows

package desktop

type noopStatusBarController struct{}

func NewStatusBarController() StatusBarController {
	return noopStatusBarController{}
}

func (noopStatusBarController) Init(func(StatusBarEvent)) {}
func (noopStatusBarController) Update(string, string, string, bool, float64, string) {
}
func (noopStatusBarController) SetLocale(StatusBarLocaleStrings) {}
func (noopStatusBarController) Destroy()                         {}
