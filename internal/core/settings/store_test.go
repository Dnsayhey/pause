package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreCreatesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := OpenSettingsStore(path)
	if err != nil {
		t.Fatalf("OpenSettingsStore() error = %v", err)
	}

	got := store.Get()
	if !got.Enforcement.OverlaySkipAllowed {
		t.Fatalf("expected overlay skip allowed by default")
	}
	if got.UI.Theme != UIThemeAuto {
		t.Fatalf("expected default theme %q, got %q", UIThemeAuto, got.UI.Theme)
	}
	if !store.WasCreated() {
		t.Fatalf("expected WasCreated()=true when config file is newly created")
	}
}

func TestStoreUpdatePersistsSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := OpenSettingsStore(path)
	if err != nil {
		t.Fatalf("OpenSettingsStore() error = %v", err)
	}

	off := false
	_, err = store.Update(SettingsPatch{
		GlobalEnabled: &off,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	reloaded, err := OpenSettingsStore(path)
	if err != nil {
		t.Fatalf("OpenSettingsStore(reload) error = %v", err)
	}
	if reloaded.WasCreated() {
		t.Fatalf("expected WasCreated()=false for existing config file")
	}
	got := reloaded.Get()
	if got.GlobalEnabled {
		t.Fatalf("expected GlobalEnabled=false after reload")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(raw), "\"globalEnabled\": false") {
		t.Fatalf("expected settings file to persist updated value, got %s", string(raw))
	}
}

func TestStoreRecoversFromCorruptedConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte("{bad json"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store, err := OpenSettingsStore(path)
	if err != nil {
		t.Fatalf("OpenSettingsStore() error = %v", err)
	}

	got := store.Get()
	want := DefaultSettings()
	if got.UI.Theme != want.UI.Theme {
		t.Fatalf("expected default theme %q after recovery, got %q", want.UI.Theme, got.UI.Theme)
	}

	matches, err := filepath.Glob(filepath.Join(dir, "settings.json.corrupt.*.bak"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 backup file, got %d", len(matches))
	}
}

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
