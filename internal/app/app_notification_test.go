package app

import (
	"testing"
	"time"

	"pause/internal/backend/runtime/state"
)

func TestBuildBreakNotificationBodyUsesRuntimeReminderNames(t *testing.T) {
	state := state.RuntimeState{
		CurrentSession: &state.BreakSessionView{
			Reasons:      []int64{1, 2},
			RemainingSec: 20,
		},
		Reminders: []state.ReminderRuntime{
			{ID: 1, Name: "护眼"},
			{ID: 2, Name: "站立"},
		},
	}

	got := buildBreakNotificationBody(state)
	want := "护眼 + 站立 break for " + (20 * time.Second).String()
	if got != want {
		t.Fatalf("buildBreakNotificationBody() = %q, want %q", got, want)
	}
}
