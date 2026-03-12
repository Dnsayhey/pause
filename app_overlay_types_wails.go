//go:build wails

package main

type breakOverlayController interface {
	Init(onSkip func())
	Show(allowSkip bool, skipButtonTitle string, countdownText string, theme string)
	Hide()
	Destroy()
	IsNative() bool
}
