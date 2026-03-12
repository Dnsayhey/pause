package config

import (
	"path/filepath"
	"testing"
)

func TestStoreCreatesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	got := store.Get()
	want := DefaultSettings()
	if got.Eye.IntervalSec != want.Eye.IntervalSec {
		t.Fatalf("expected default eye interval %d, got %d", want.Eye.IntervalSec, got.Eye.IntervalSec)
	}
	if !got.Enforcement.OverlayEnabled {
		t.Fatalf("expected overlay enabled by default")
	}
}

func TestStoreUpdatePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	off := false
	interval := 1500
	_, err = store.Update(SettingsPatch{
		GlobalEnabled: &off,
		Eye: &ReminderRulePatch{
			IntervalSec: &interval,
		},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	reloaded, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore(reload) error = %v", err)
	}
	got := reloaded.Get()
	if got.GlobalEnabled {
		t.Fatalf("expected GlobalEnabled=false after reload")
	}
	if got.Eye.IntervalSec != interval {
		t.Fatalf("expected Eye.IntervalSec=%d after reload, got %d", interval, got.Eye.IntervalSec)
	}
}

func TestApplyPatchNormalizesValues(t *testing.T) {
	base := DefaultSettings()
	badMode := "bad_mode"
	badVolume := 999
	badThreshold := -1
	badLanguage := "fr-FR"

	got := base.ApplyPatch(SettingsPatch{
		Sound: &SoundSettingsPatch{Volume: &badVolume},
		Timer: &TimerSettingsPatch{
			Mode:                  &badMode,
			IdlePauseThresholdSec: &badThreshold,
		},
		UI: &UISettingsPatch{
			Language: &badLanguage,
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
