package app

import "pause/internal/core/settings"

func resolveEffectiveTheme(setting string) string {
	return resolveEffectiveThemeWithSystem(setting, detectPreferredTheme())
}

func resolveEffectiveThemeWithSystem(setting string, systemTheme string) string {
	normalized := settings.NormalizeUITheme(setting)
	if normalized == settings.UIThemeLight || normalized == settings.UIThemeDark {
		return normalized
	}

	system := settings.NormalizeUITheme(systemTheme)
	if system == settings.UIThemeLight || system == settings.UIThemeDark {
		return system
	}

	return settings.UIThemeDark
}
