//go:build !wails || (!darwin && !windows)

package app

func detectPreferredLanguage() string {
	return ""
}
