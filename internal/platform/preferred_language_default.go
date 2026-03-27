//go:build !wails || (!darwin && !windows)

package platform

func DetectPreferredLanguage() string {
	return ""
}
