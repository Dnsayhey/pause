package settings

import "strings"

const (
	UILanguageAuto = "auto"
	UILanguageZhCN = "zh-CN"
	UILanguageEnUS = "en-US"
)

func NormalizeUILanguage(value string) string {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return UILanguageAuto
	}

	lower := strings.ToLower(cleaned)
	switch lower {
	case "auto":
		return UILanguageAuto
	case "zh", "zh-cn", "zh_cn", "zh-hans", "zh-hant":
		return UILanguageZhCN
	case "en", "en-us", "en_us":
		return UILanguageEnUS
	default:
		return UILanguageAuto
	}
}

func LanguageFromLocaleValue(value string) string {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return ""
	}
	lower := strings.ToLower(cleaned)
	if strings.HasPrefix(lower, "zh") {
		return UILanguageZhCN
	}
	if strings.HasPrefix(lower, "en") {
		return UILanguageEnUS
	}
	return ""
}

func ResolveEffectiveUILanguage(setting string, preferredLocales ...string) string {
	normalized := NormalizeUILanguage(setting)
	if normalized != UILanguageAuto {
		return normalized
	}
	for _, locale := range preferredLocales {
		if preferred := LanguageFromLocaleValue(locale); preferred != "" {
			return preferred
		}
	}
	return UILanguageEnUS
}
