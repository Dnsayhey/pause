//go:build !darwin || !wails

package main

func detectPreferredTheme() string {
	return ""
}
