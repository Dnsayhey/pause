//go:build !wails || (!darwin && !windows)

package app

func detectPreferredTheme() string {
	return ""
}
