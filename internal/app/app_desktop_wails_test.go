//go:build wails

package app

import (
	"testing"

	"pause/internal/core/config"
	"pause/internal/core/service"
)

func TestOverlaySkipMode_AllowSkipUsesNormal(t *testing.T) {
	settings := config.DefaultSettings()
	settings.Enforcement.OverlaySkipAllowed = true

	if got := overlaySkipMode(settings); got != service.SkipModeNormal {
		t.Fatalf("overlaySkipMode() = %q, want %q", got, service.SkipModeNormal)
	}
}

func TestOverlaySkipMode_DisallowSkipUsesEmergency(t *testing.T) {
	settings := config.DefaultSettings()
	settings.Enforcement.OverlaySkipAllowed = false

	if got := overlaySkipMode(settings); got != service.SkipModeEmergency {
		t.Fatalf("overlaySkipMode() = %q, want %q", got, service.SkipModeEmergency)
	}
}

func TestBuildCountdownLabel_MultiReminderOrder(t *testing.T) {
	state := config.RuntimeState{
		GlobalEnabled: true,
		Reminders: []config.ReminderRuntime{
			{ID: config.ReminderIDEye, Enabled: true, NextInSec: 300, IntervalSec: 1200},
			{ID: config.ReminderIDStand, Enabled: true, NextInSec: 120, IntervalSec: 3600},
		},
	}

	got := buildCountdownLabel(state, config.UILanguageZhCN)
	want := "站立 - 02:00\n护眼 - 05:00"
	if got != want {
		t.Fatalf("buildCountdownLabel() = %q, want %q", got, want)
	}
}

func TestBuildCountdownLabel_OffFallback(t *testing.T) {
	state := config.RuntimeState{}

	gotZh := buildCountdownLabel(state, config.UILanguageZhCN)
	if gotZh != "暂无提醒" {
		t.Fatalf("buildCountdownLabel() zh = %q, want %q", gotZh, "暂无提醒")
	}

	gotEn := buildCountdownLabel(state, config.UILanguageEnUS)
	if gotEn != "No reminders" {
		t.Fatalf("buildCountdownLabel() en = %q, want %q", gotEn, "No reminders")
	}
}

func TestBuildCountdownLabel_Paused(t *testing.T) {
	state := config.RuntimeState{
		GlobalEnabled: false,
		Reminders: []config.ReminderRuntime{
			{ID: config.ReminderIDEye, Enabled: true, NextInSec: 300, IntervalSec: 1200},
			{ID: config.ReminderIDStand, Enabled: true, NextInSec: 120, IntervalSec: 3600},
		},
	}

	got := buildCountdownLabel(state, config.UILanguageZhCN)
	want := "站立 - 已暂停\n护眼 - 已暂停"
	if got != want {
		t.Fatalf("buildCountdownLabel() paused = %q, want %q", got, want)
	}
}

func TestBuildCountdownLabel_OnlyRestTypeReminders(t *testing.T) {
	state := config.RuntimeState{
		GlobalEnabled: true,
		Reminders: []config.ReminderRuntime{
			{ID: "notify-1", Name: "通知提醒", ReminderType: "notify", Enabled: true, NextInSec: 60, IntervalSec: 600},
			{ID: "rest-1", Name: "喝水", ReminderType: "rest", Enabled: true, NextInSec: 120, IntervalSec: 1200},
		},
	}

	got := buildCountdownLabel(state, config.UILanguageZhCN)
	want := "喝水 - 02:00"
	if got != want {
		t.Fatalf("buildCountdownLabel() = %q, want %q", got, want)
	}
}

func TestSelectAutoReminderChoice_SkipsNotifyReminder(t *testing.T) {
	state := config.RuntimeState{
		GlobalEnabled: true,
		Reminders: []config.ReminderRuntime{
			{ID: "notify-1", ReminderType: "notify", Enabled: true, NextInSec: 10, IntervalSec: 600},
			{ID: "rest-1", ReminderType: "rest", Enabled: true, NextInSec: 20, IntervalSec: 1200},
		},
	}

	choice := selectAutoReminderChoice(state)
	if choice.reason != "rest-1" {
		t.Fatalf("selectAutoReminderChoice() reason = %q, want %q", choice.reason, "rest-1")
	}
}
