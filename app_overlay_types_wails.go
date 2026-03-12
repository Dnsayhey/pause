//go:build wails

package main

type breakOverlayController interface {
	Init(onSkip func())
	Show(allowSkip bool, skipButtonTitle string, countdownText string)
	Hide()
	Destroy()
	IsNative() bool
}
