//go:build wails

package app

import (
	"testing"

	"pause/internal/core/config"
)

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
