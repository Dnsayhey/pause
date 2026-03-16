//go:build wails && darwin

package desktop

func SupportsBackgroundHideOnClose() bool {
	return true
}
