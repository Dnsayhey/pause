//go:build !darwin || !wails

package app

func detectPreferredLanguage() string {
	return ""
}
