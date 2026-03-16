//go:build wails && windows

package desktop

func SupportsBackgroundHideOnClose() bool {
	return true
}
