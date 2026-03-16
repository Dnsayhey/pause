//go:build wails && !darwin && !windows

package desktop

func SupportsBackgroundHideOnClose() bool {
	return false
}
