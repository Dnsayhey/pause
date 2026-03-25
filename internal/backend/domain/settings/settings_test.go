package settings

import "testing"

func TestApplyPatchNormalizesValues(t *testing.T) {
	base := DefaultSettings()
	badMode := "bad_mode"
	badVolume := 999
	badThreshold := -1
	badLanguage := "fr-FR"
	badTheme := "sepia"

	got := base.ApplyPatch(SettingsPatch{
		Sound: &SoundSettingsPatch{Volume: &badVolume},
		Timer: &TimerSettingsPatch{
			Mode:                  &badMode,
			IdlePauseThresholdSec: &badThreshold,
		},
		UI: &UISettingsPatch{
			Language: &badLanguage,
			Theme:    &badTheme,
		},
	})

	if got.Timer.Mode != TimerModeIdlePause {
		t.Fatalf("expected timer mode fallback to %q, got %q", TimerModeIdlePause, got.Timer.Mode)
	}
	if got.Sound.Volume != base.Sound.Volume {
		t.Fatalf("expected volume fallback to %d, got %d", base.Sound.Volume, got.Sound.Volume)
	}
	if got.Timer.IdlePauseThresholdSec != base.Timer.IdlePauseThresholdSec {
		t.Fatalf("expected idle threshold fallback to %d, got %d", base.Timer.IdlePauseThresholdSec, got.Timer.IdlePauseThresholdSec)
	}
	if got.UI.Language != base.UI.Language {
		t.Fatalf("expected language fallback to %q, got %q", base.UI.Language, got.UI.Language)
	}
	if got.UI.Theme != base.UI.Theme {
		t.Fatalf("expected theme fallback to %q, got %q", base.UI.Theme, got.UI.Theme)
	}
}

func TestNormalizeUILanguage(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{input: "", want: UILanguageAuto},
		{input: "auto", want: UILanguageAuto},
		{input: "zh", want: UILanguageZhCN},
		{input: "zh-CN", want: UILanguageZhCN},
		{input: "en", want: UILanguageEnUS},
		{input: "en-US", want: UILanguageEnUS},
		{input: "unknown", want: UILanguageAuto},
	}

	for _, tc := range cases {
		got := NormalizeUILanguage(tc.input)
		if got != tc.want {
			t.Fatalf("NormalizeUILanguage(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestNormalizeUITheme(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{input: "", want: UIThemeAuto},
		{input: "auto", want: UIThemeAuto},
		{input: "light", want: UIThemeLight},
		{input: "dark", want: UIThemeDark},
		{input: "unknown", want: UIThemeAuto},
	}

	for _, tc := range cases {
		got := NormalizeUITheme(tc.input)
		if got != tc.want {
			t.Fatalf("NormalizeUITheme(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
