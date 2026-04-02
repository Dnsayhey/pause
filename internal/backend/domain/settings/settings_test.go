package settings

import "testing"

func TestSettingsApplyPatch_NormalizeInvalidValues(t *testing.T) {
	base := DefaultSettings()
	badMode := "wrong"
	badThreshold := -3
	badLanguage := "ja-JP"
	badTheme := "sepia"

	got := base.ApplyPatch(SettingsPatch{
		Timer: &TimerSettingsPatch{Mode: &badMode, IdlePauseThresholdSec: &badThreshold},
		UI:    &UISettingsPatch{Language: &badLanguage, Theme: &badTheme},
	})

	if got.Timer.Mode != TimerModeIdlePause {
		t.Fatalf("timer mode fallback mismatch: got=%q", got.Timer.Mode)
	}
	if got.Timer.IdlePauseThresholdSec != base.Timer.IdlePauseThresholdSec {
		t.Fatalf("idle threshold fallback mismatch: got=%d want=%d", got.Timer.IdlePauseThresholdSec, base.Timer.IdlePauseThresholdSec)
	}
	if got.UI.Language != base.UI.Language {
		t.Fatalf("language fallback mismatch: got=%q want=%q", got.UI.Language, base.UI.Language)
	}
	if got.UI.Theme != base.UI.Theme {
		t.Fatalf("theme fallback mismatch: got=%q want=%q", got.UI.Theme, base.UI.Theme)
	}
}

func TestNormalizeUILanguage_Table(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", UILanguageAuto},
		{"auto", UILanguageAuto},
		{"zh", UILanguageZhCN},
		{"zh-CN", UILanguageZhCN},
		{"en", UILanguageEnUS},
		{"en-US", UILanguageEnUS},
		{"unknown", UILanguageAuto},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := NormalizeUILanguage(tc.in); got != tc.want {
				t.Fatalf("NormalizeUILanguage(%q)=%q want=%q", tc.in, got, tc.want)
			}
		})
	}
}

func TestResolveEffectiveUILanguage_Auto(t *testing.T) {
	cases := []struct {
		name    string
		setting string
		locales []string
		want    string
	}{
		{name: "explicit zh", setting: UILanguageZhCN, locales: []string{"en-US"}, want: UILanguageZhCN},
		{name: "auto zh locale", setting: UILanguageAuto, locales: []string{"zh-Hans-CN"}, want: UILanguageZhCN},
		{name: "auto en locale", setting: UILanguageAuto, locales: []string{"en-US"}, want: UILanguageEnUS},
		{name: "auto fallback", setting: UILanguageAuto, locales: []string{"fr-FR"}, want: UILanguageEnUS},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveEffectiveUILanguage(tc.setting, tc.locales...); got != tc.want {
				t.Fatalf("ResolveEffectiveUILanguage(%q, %v)=%q want=%q", tc.setting, tc.locales, got, tc.want)
			}
		})
	}
}

func TestNormalizeUITheme_Table(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", UIThemeAuto},
		{"auto", UIThemeAuto},
		{"light", UIThemeLight},
		{"dark", UIThemeDark},
		{"unknown", UIThemeAuto},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := NormalizeUITheme(tc.in); got != tc.want {
				t.Fatalf("NormalizeUITheme(%q)=%q want=%q", tc.in, got, tc.want)
			}
		})
	}
}
