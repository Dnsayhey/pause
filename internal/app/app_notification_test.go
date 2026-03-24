package app

import (
	"testing"
	"time"

	"pause/internal/core/config"
)

func TestBuildBreakNotificationBodyUsesRuntimeReminderNames(t *testing.T) {
	state := config.RuntimeState{
		CurrentSession: &config.BreakSessionView{
			Reasons:      []string{"eye", "stand"},
			RemainingSec: 20,
		},
		Reminders: []config.ReminderRuntime{
			{ID: "eye", Name: "护眼"},
			{ID: "stand", Name: "站立"},
		},
	}

	got := buildBreakNotificationBody(state)
	want := "护眼 + 站立 break for " + (20 * time.Second).String()
	if got != want {
		t.Fatalf("buildBreakNotificationBody() = %q, want %q", got, want)
	}
}
