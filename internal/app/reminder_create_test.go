package app

import (
	"path/filepath"
	"testing"

	"pause/internal/core/config"
	"pause/internal/core/history"
)

func TestNormalizeReminderCreateInput(t *testing.T) {
	valid, err := normalizeReminderCreateInput(config.ReminderCreateInput{
		Name:        "  Focus  ",
		IntervalSec: 1500,
		BreakSec:    30,
	})
	if err != nil {
		t.Fatalf("normalizeReminderCreateInput(valid) error = %v", err)
	}
	if valid.Name != "Focus" {
		t.Fatalf("expected trimmed name Focus, got %q", valid.Name)
	}

	if _, err := normalizeReminderCreateInput(config.ReminderCreateInput{IntervalSec: 1, BreakSec: 1}); err == nil {
		t.Fatalf("expected missing name to fail")
	}
	if _, err := normalizeReminderCreateInput(config.ReminderCreateInput{Name: "x", IntervalSec: 0, BreakSec: 1}); err == nil {
		t.Fatalf("expected non-positive interval to fail")
	}
	if _, err := normalizeReminderCreateInput(config.ReminderCreateInput{Name: "x", IntervalSec: 1, BreakSec: 0}); err == nil {
		t.Fatalf("expected non-positive break to fail")
	}
}

func TestReminderIDBaseFromName(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{name: "Focus Time", want: "focus-time"},
		{name: "focus_time.v2", want: "focus-time-v2"},
		{name: "  __  ", want: "reminder"},
		{name: "喝水提醒", want: "reminder"},
	}
	for _, tc := range cases {
		got := reminderIDBaseFromName(tc.name)
		if got != tc.want {
			t.Fatalf("reminderIDBaseFromName(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestCreateReminderInHistoryGeneratesUniqueIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := history.OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	input := config.ReminderCreateInput{
		Name:        "Focus Time",
		IntervalSec: 1500,
		BreakSec:    30,
	}

	id1, err := createReminderInHistory(store, input)
	if err != nil {
		t.Fatalf("createReminderInHistory(first) error = %v", err)
	}
	if id1 != "focus-time" {
		t.Fatalf("expected first id focus-time, got %q", id1)
	}

	id2, err := createReminderInHistory(store, input)
	if err != nil {
		t.Fatalf("createReminderInHistory(second) error = %v", err)
	}
	if id2 != "focus-time-2" {
		t.Fatalf("expected second id focus-time-2, got %q", id2)
	}
}
