//go:build wails

package app

import (
	"testing"

	"pause/internal/backend/runtime/state"
	"pause/internal/core/settings"
)

const (
	testReminderIDEye   int64 = 1
	testReminderIDStand int64 = 2
)

func TestOverlaySkipMode_AllowSkipUsesNormal(t *testing.T) {
	settings := settings.DefaultSettings()
	settings.Enforcement.OverlaySkipAllowed = true

	if got := overlaySkipMode(settings); got != skipModeNormal {
		t.Fatalf("overlaySkipMode() = %q, want %q", got, skipModeNormal)
	}
}

func TestOverlaySkipMode_DisallowSkipUsesEmergency(t *testing.T) {
	settings := settings.DefaultSettings()
	settings.Enforcement.OverlaySkipAllowed = false

	if got := overlaySkipMode(settings); got != skipModeEmergency {
		t.Fatalf("overlaySkipMode() = %q, want %q", got, skipModeEmergency)
	}
}

func TestBuildCountdownLabel_MultiReminderOrder(t *testing.T) {
	state := state.RuntimeState{
		GlobalEnabled: true,
		Reminders: []state.ReminderRuntime{
			{ID: testReminderIDEye, Name: "护眼", Enabled: true, NextInSec: 300, IntervalSec: 1200},
			{ID: testReminderIDStand, Name: "站立", Enabled: true, NextInSec: 120, IntervalSec: 3600},
		},
	}

	got := buildCountdownLabel(state, settings.UILanguageZhCN)
	want := "站立 - 02:00\n护眼 - 05:00"
	if got != want {
		t.Fatalf("buildCountdownLabel() = %q, want %q", got, want)
	}
}

func TestBuildCountdownLabel_OffFallback(t *testing.T) {
	state := state.RuntimeState{}

	gotZh := buildCountdownLabel(state, settings.UILanguageZhCN)
	if gotZh != "暂无提醒" {
		t.Fatalf("buildCountdownLabel() zh = %q, want %q", gotZh, "暂无提醒")
	}

	gotEn := buildCountdownLabel(state, settings.UILanguageEnUS)
	if gotEn != "No reminders" {
		t.Fatalf("buildCountdownLabel() en = %q, want %q", gotEn, "No reminders")
	}
}

func TestBuildCountdownLabel_Paused(t *testing.T) {
	state := state.RuntimeState{
		GlobalEnabled: false,
		Reminders: []state.ReminderRuntime{
			{ID: testReminderIDEye, Name: "护眼", Enabled: true, NextInSec: 300, IntervalSec: 1200},
			{ID: testReminderIDStand, Name: "站立", Enabled: true, NextInSec: 120, IntervalSec: 3600},
		},
	}

	got := buildCountdownLabel(state, settings.UILanguageZhCN)
	want := "站立 - 已暂停\n护眼 - 已暂停"
	if got != want {
		t.Fatalf("buildCountdownLabel() paused = %q, want %q", got, want)
	}
}

func TestBuildCountdownLabel_OnlyRestTypeReminders(t *testing.T) {
	state := state.RuntimeState{
		GlobalEnabled: true,
		Reminders: []state.ReminderRuntime{
			{ID: 1, Name: "通知提醒", ReminderType: "notify", Enabled: true, NextInSec: 60, IntervalSec: 600},
			{ID: 2, Name: "喝水", ReminderType: "rest", Enabled: true, NextInSec: 120, IntervalSec: 1200},
		},
	}

	got := buildCountdownLabel(state, settings.UILanguageZhCN)
	want := "喝水 - 02:00"
	if got != want {
		t.Fatalf("buildCountdownLabel() = %q, want %q", got, want)
	}
}

func TestSelectAutoReminderChoice_SkipsNotifyReminder(t *testing.T) {
	state := state.RuntimeState{
		GlobalEnabled: true,
		Reminders: []state.ReminderRuntime{
			{ID: 1, ReminderType: "notify", Enabled: true, NextInSec: 10, IntervalSec: 600},
			{ID: 2, ReminderType: "rest", Enabled: true, NextInSec: 20, IntervalSec: 1200},
		},
	}

	choice := selectAutoReminderChoice(state)
	if choice.reason != 2 {
		t.Fatalf("selectAutoReminderChoice() reason = %d, want %d", choice.reason, 2)
	}
}
