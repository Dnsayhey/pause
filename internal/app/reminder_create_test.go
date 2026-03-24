package app

import (
	"path/filepath"
	"testing"

	"pause/internal/core/config"
	"pause/internal/core/history"
)

func TestNormalizeReminderCreateInput(t *testing.T) {
	notify := " Notify "
	valid, err := normalizeReminderCreateInput(config.ReminderCreateInput{
		Name:         "  Focus  ",
		IntervalSec:  1500,
		BreakSec:     30,
		ReminderType: &notify,
	})
	if err != nil {
		t.Fatalf("normalizeReminderCreateInput(valid) error = %v", err)
	}
	if valid.Name != "Focus" {
		t.Fatalf("expected trimmed name Focus, got %q", valid.Name)
	}
	if valid.ReminderType == nil || *valid.ReminderType != "notify" {
		t.Fatalf("expected normalized reminder type notify, got %#v", valid.ReminderType)
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
	invalidType := "unknown"
	if _, err := normalizeReminderCreateInput(config.ReminderCreateInput{
		Name:         "x",
		IntervalSec:  1,
		BreakSec:     1,
		ReminderType: &invalidType,
	}); err == nil {
		t.Fatalf("expected invalid reminderType to fail")
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

func TestApplyReminderPatchToHistoryRejectsUnknownReminderID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := history.OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	if err := store.CreateReminder(history.ReminderMutation{ID: "eye"}); err != nil {
		t.Fatalf("CreateReminder(eye) error = %v", err)
	}

	enabled := true
	patches := []config.ReminderPatch{{ID: "missing", Enabled: &enabled}}
	if err := applyReminderPatchToHistory(store, patches); err == nil {
		t.Fatalf("expected unknown reminder id patch to fail")
	}
}

func TestApplyReminderPatchToHistoryRejectsInvalidPatchValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := history.OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	if err := store.CreateReminder(history.ReminderMutation{ID: "eye"}); err != nil {
		t.Fatalf("CreateReminder(eye) error = %v", err)
	}

	zero := 0
	invalidType := "x"
	patches := []config.ReminderPatch{
		{ID: "eye", IntervalSec: &zero},
	}
	if err := applyReminderPatchToHistory(store, patches); err == nil {
		t.Fatalf("expected invalid interval patch to fail")
	}
	patches = []config.ReminderPatch{
		{ID: "eye", ReminderType: &invalidType},
	}
	if err := applyReminderPatchToHistory(store, patches); err == nil {
		t.Fatalf("expected invalid reminder type patch to fail")
	}
}
