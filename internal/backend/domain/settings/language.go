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
