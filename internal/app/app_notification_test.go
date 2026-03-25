package app

import (
	"testing"
	"time"

	"pause/internal/backend/runtime/state"
)

func TestBuildBreakNotificationBody_WithReasonNames(t *testing.T) {
	rs := state.RuntimeState{
		CurrentSession: &state.BreakSessionView{Reasons: []int64{1, 2}, RemainingSec: 20},
		Reminders: []state.ReminderRuntime{
			{ID: 1, Name: "护眼"},
			{ID: 2, Name: "站立"},
		},
	}

	got := buildBreakNotificationBody(rs)
	want := "护眼 + 站立 break for " + (20 * time.Second).String()
	if got != want {
		t.Fatalf("buildBreakNotificationBody()=%q want=%q", got, want)
	}
}

func TestBuildBreakNotificationBody_WithoutSession(t *testing.T) {
	if got := buildBreakNotificationBody(state.RuntimeState{}); got != "Break started" {
		t.Fatalf("expected fallback message, got=%q", got)
	}
}
