package app

import (
	"fmt"
	"os"
	"strings"

	"pause/internal/backend/domain/settings"
	"pause/internal/desktop"
	"pause/internal/platform"
)

func resolveEffectiveLanguage(setting string) string {
	preferredLocales := []string{platform.DetectPreferredLanguage()}
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		preferredLocales = append(preferredLocales, os.Getenv(key))
	}
	return settings.ResolveEffectiveUILanguage(setting, preferredLocales...)
}

func buildStatusBarLocaleStrings(language string) desktop.StatusBarLocaleStrings {
	if language == settings.UILanguageZhCN {
		return desktop.StatusBarLocaleStrings{
			PopoverTitle:          "Pause",
			BreakNowButton:        "立即休息",
			PauseButton:           "暂停计时",
			ResumeButton:          "恢复计时",
			OpenAppButton:         "打开主界面",
			AboutMenuItem:         "关于",
			QuitMenuItem:          "退出",
			MoreButtonTip:         "更多",
			Tooltip:               "Pause 休息提醒",
			StatusLineFallback:    "运行状态：--",
			NextBreakLineFallback: "下一次休息：--:--",
		}
	}
	return desktop.StatusBarLocaleStrings{
		PopoverTitle:          "Pause",
		BreakNowButton:        "Break now",
		PauseButton:           "Pause timer",
		ResumeButton:          "Resume timer",
		OpenAppButton:         "Open Main Window",
		AboutMenuItem:         "About",
		QuitMenuItem:          "Quit",
		MoreButtonTip:         "More",
		Tooltip:               "Pause break reminder",
		StatusLineFallback:    "Status: --",
		NextBreakLineFallback: "Next break: --:--",
	}
}

func localizedBuiltInReminderSeedNames(language string) (eye string, stand string, water string) {
	normalized := settings.NormalizeUILanguage(strings.TrimSpace(language))
	switch normalized {
	case settings.UILanguageZhCN:
		return "护眼", "站立", "喝水"
	default:
		return "Eye", "Stand", "Hydrate"
	}
}

func overlaySkipButtonTitle(language string) string {
	if language == settings.UILanguageZhCN {
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

func overlayMessageText(language string) string {
	if language == settings.UILanguageZhCN {
		return "短暂休息，是为了更长时间地专注。"
	}
	return "A short break helps you stay focused longer."
}

func formatOverlayCountdown(sec int) string {
	if sec < 0 {
		sec = 0
	}
	m := sec / 60
	s := sec % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}
