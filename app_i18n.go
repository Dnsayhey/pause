package main

import (
	"fmt"
	"os"
	"strings"

	"pause/internal/config"
)

type statusBarLocaleStrings struct {
	PopoverTitle   string
	BreakNowButton string
	PauseButton    string
	Pause30Button  string
	ResumeButton   string
	OpenAppButton  string
	AboutMenuItem  string
	QuitMenuItem   string
	MoreButtonTip  string
	Tooltip        string
}

func resolveEffectiveLanguage(setting string) string {
	normalized := config.NormalizeUILanguage(setting)
	if normalized != config.UILanguageAuto {
		return normalized
	}

	if preferred := languageFromLocaleValue(detectPreferredLanguage()); preferred != "" {
		return preferred
	}

	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if lang := languageFromLocaleValue(os.Getenv(key)); lang != "" {
			return lang
		}
	}
	return config.UILanguageEnUS
}

func languageFromLocaleValue(value string) string {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return ""
	}
	lower := strings.ToLower(cleaned)
	if strings.HasPrefix(lower, "zh") {
		return config.UILanguageZhCN
	}
	if strings.HasPrefix(lower, "en") {
		return config.UILanguageEnUS
	}
	return ""
}

func buildStatusBarLocaleStrings(language string) statusBarLocaleStrings {
	if language == config.UILanguageZhCN {
		return statusBarLocaleStrings{
			PopoverTitle:   "Pause",
			BreakNowButton: "zzZ",
			PauseButton:    "暂停",
			Pause30Button:  "暂停 30 分钟",
			ResumeButton:   "恢复",
			OpenAppButton:  "打开主界面",
			AboutMenuItem:  "关于",
			QuitMenuItem:   "退出",
			MoreButtonTip:  "更多",
			Tooltip:        "Pause 休息提醒",
		}
	}
	return statusBarLocaleStrings{
		PopoverTitle:   "Pause",
		BreakNowButton: "zzZ",
		PauseButton:    "Pause",
		Pause30Button:  "Pause 30m",
		ResumeButton:   "Resume",
		OpenAppButton:  "Open Pause",
		AboutMenuItem:  "About",
		QuitMenuItem:   "Quit",
		MoreButtonTip:  "More",
		Tooltip:        "Pause break reminder",
	}
}

func localizeReminderReason(reason string, language string) string {
	switch reason {
	case "eye":
		if language == config.UILanguageZhCN {
			return "护眼"
		}
		return "eye"
	case "stand":
		if language == config.UILanguageZhCN {
			return "站立"
		}
		return "stand"
	default:
		return reason
	}
}

func overlaySkipButtonTitle(language string) string {
	if language == config.UILanguageZhCN {
		return "跳过"
	}
	return "Skip"
}

func overlayCountdownText(language string, remainingSec int) string {
	if remainingSec < 0 {
		remainingSec = 0
	}
	return formatOverlayCountdown(remainingSec)
}

func formatOverlayCountdown(sec int) string {
	if sec < 0 {
		sec = 0
	}
	m := sec / 60
	s := sec % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}
