package main

import "pause/internal/config"

func resolveEffectiveTheme(setting string) string {
	return resolveEffectiveThemeWithSystem(setting, detectPreferredTheme())
}

func resolveEffectiveThemeWithSystem(setting string, systemTheme string) string {
	normalized := config.NormalizeUITheme(setting)
	if normalized == config.UIThemeLight || normalized == config.UIThemeDark {
		return normalized
	}

	system := config.NormalizeUITheme(systemTheme)
	if system == config.UIThemeLight || system == config.UIThemeDark {
		return system
	}

	return config.UIThemeDark
}
