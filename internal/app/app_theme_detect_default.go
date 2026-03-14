//go:build !darwin || !wails

package app

func detectPreferredTheme() string {
	return ""
}
