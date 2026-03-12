//go:build !darwin || !wails

package main

func detectPreferredLanguage() string {
	return ""
}
