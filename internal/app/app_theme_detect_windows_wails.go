//go:build wails && windows

package app

import "golang.org/x/sys/windows/registry"

const personalizeKeyPath = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`

func detectPreferredTheme() string {
	key, err := registry.OpenKey(registry.CURRENT_USER, personalizeKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer key.Close()

	// AppsUseLightTheme: 1 = light, 0 = dark.
	v, _, err := key.GetIntegerValue("AppsUseLightTheme")
	if err != nil {
		return ""
	}
	if v == 0 {
		return "dark"
	}
	return "light"
}

