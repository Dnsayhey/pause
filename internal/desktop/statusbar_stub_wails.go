//go:build wails && !darwin

package desktop

type noopStatusBarController struct{}

func NewStatusBarController() StatusBarController {
	return noopStatusBarController{}
}

func (noopStatusBarController) Init(func(int))                               {}
func (noopStatusBarController) Update(string, string, string, bool, float64) {}
func (noopStatusBarController) SetLocale(StatusBarLocaleStrings)             {}
func (noopStatusBarController) Destroy()                                     {}
