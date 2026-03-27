package engine

import (
	"os"

	"pause/internal/backend/domain/settings"
	"pause/internal/platform"
)

func resolveEffectiveLanguage(setting string) string {
	return settings.ResolveEffectiveUILanguage(
		setting,
		platform.DetectPreferredLanguage(),
		os.Getenv("LC_ALL"),
		os.Getenv("LC_MESSAGES"),
		os.Getenv("LANG"),
	)
}
