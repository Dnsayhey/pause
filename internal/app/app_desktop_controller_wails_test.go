//go:build wails

package app

import (
	"testing"

	"pause/internal/backend/domain/settings"
	"pause/internal/backend/runtime/state"
)

func TestOverlaySkipMode(t *testing.T) {
	cfg := settings.DefaultSettings()
	cfg.Enforcement.OverlaySkipAllowed = true
	if got := overlaySkipMode(cfg); got != skipModeNormal {
		t.Fatalf("overlaySkipMode(true)=%q want=%q", got, skipModeNormal)
	}
	cfg.Enforcement.OverlaySkipAllowed = false
	if got := overlaySkipMode(cfg); got != skipModeEmergency {
		t.Fatalf("overlaySkipMode(false)=%q want=%q", got, skipModeEmergency)
	}
}

func TestBuildCountdownLabel(t *testing.T) {
	rs := state.RuntimeState{
		GlobalEnabled: true,
		Reminders: []state.ReminderRuntime{
			{ID: 1, Name: "护眼", ReminderType: "rest", Enabled: true, NextInSec: 300, IntervalSec: 1200},
			{ID: 2, Name: "站立", ReminderType: "rest", Enabled: true, NextInSec: 120, IntervalSec: 3600},
			{ID: 3, Name: "喝水", ReminderType: "notify", Enabled: true, NextInSec: 60, IntervalSec: 600},
		},
	}
	if got := buildCountdownLabel(rs, settings.UILanguageZhCN); got != "站立 - 02:00\n护眼 - 05:00" {
		t.Fatalf("unexpected countdown label: %q", got)
	}
}

func TestSelectAutoReminderChoiceSkipsNotify(t *testing.T) {
	rs := state.RuntimeState{
		GlobalEnabled: true,
		Reminders: []state.ReminderRuntime{
			{ID: 1, ReminderType: "notify", Enabled: true, NextInSec: 10, IntervalSec: 600},
			{ID: 2, ReminderType: "rest", Enabled: true, NextInSec: 20, IntervalSec: 1200},
		},
	}
	choice := selectAutoReminderChoice(rs)
	if choice.reason != 2 {
		t.Fatalf("expected rest reminder selected, got=%d", choice.reason)
	}
}
