//go:build wails && !darwin

package main

type statusBarController interface {
	Init(onAction func(actionID int))
	Update(status, countdown, title string, paused bool, progress float64)
	SetLocale(strings statusBarLocaleStrings)
	Destroy()
}

type noopStatusBarController struct{}

func newStatusBarController() statusBarController {
	return noopStatusBarController{}
}

func (noopStatusBarController) Init(func(int))                               {}
func (noopStatusBarController) Update(string, string, string, bool, float64) {}
func (noopStatusBarController) SetLocale(statusBarLocaleStrings)             {}
func (noopStatusBarController) Destroy()                                     {}
