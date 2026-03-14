package config

import "strings"

const (
	UIThemeAuto  = "auto"
	UIThemeLight = "light"
	UIThemeDark  = "dark"
)

func NormalizeUITheme(value string) string {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return UIThemeAuto
	}

	switch strings.ToLower(cleaned) {
	case UIThemeAuto:
		return UIThemeAuto
	case UIThemeLight:
		return UIThemeLight
	case UIThemeDark:
		return UIThemeDark
	default:
		return UIThemeAuto
	}
}
