package app

import (
	"path/filepath"
	"testing"

	"pause/internal/core/config"
	"pause/internal/core/history"
)

func TestNormalizeReminderCreateInput(t *testing.T) {
	notify := "notify"
	valid, err := normalizeReminderCreateInput(config.ReminderCreateInput{
		Name:         "Focus",
		IntervalSec:  1500,
		BreakSec:     30,
		ReminderType: &notify,
	})
	if err != nil {
		t.Fatalf("normalizeReminderCreateInput(valid) error = %v", err)
	}
	if valid.Name != "Focus" {
		t.Fatalf("expected name Focus, got %q", valid.Name)
	}
	if valid.ReminderType == nil || *valid.ReminderType != "notify" {
		t.Fatalf("expected reminder type notify, got %#v", valid.ReminderType)
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
	if _, err := normalizeReminderCreateInput(config.ReminderCreateInput{Name: "x", IntervalSec: 1, BreakSec: 1}); err == nil {
		t.Fatalf("expected missing reminder type to fail")
	}
	if _, err := normalizeReminderCreateInput(config.ReminderCreateInput{Name: " x ", IntervalSec: 1, BreakSec: 1, ReminderType: &notify}); err == nil {
		t.Fatalf("expected name with leading/trailing spaces to fail")
	}
	invalidType := "Notify"
	if _, err := normalizeReminderCreateInput(config.ReminderCreateInput{Name: "x", IntervalSec: 1, BreakSec: 1, ReminderType: &invalidType}); err == nil {
		t.Fatalf("expected invalid reminder type case to fail")
	}
}

func TestCreateReminderInHistoryUsesAutoIncrementID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := history.OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	reminderType := "rest"
	inputA := config.ReminderCreateInput{Name: "Focus Time A", IntervalSec: 1500, BreakSec: 30, ReminderType: &reminderType}
	inputB := config.ReminderCreateInput{Name: "Focus Time B", IntervalSec: 1200, BreakSec: 20, ReminderType: &reminderType}

	id1, err := createReminderInHistory(store, inputA)
	if err != nil {
		t.Fatalf("createReminderInHistory(first) error = %v", err)
	}
	if id1 <= 0 {
		t.Fatalf("expected first id > 0, got %d", id1)
	}

	id2, err := createReminderInHistory(store, inputB)
	if err != nil {
		t.Fatalf("createReminderInHistory(second) error = %v", err)
	}
	if id2 <= id1 {
		t.Fatalf("expected second id to increase, got first=%d second=%d", id1, id2)
	}
}

func TestApplyReminderPatchToHistoryRejectsUnknownReminderID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := history.OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	name := "eye"
	enabled := true
	intervalSec := 20 * 60
	breakSec := 20
	reminderType := "rest"
	if _, err := store.CreateReminder(history.ReminderMutation{
		Name:         &name,
		Enabled:      &enabled,
		IntervalSec:  &intervalSec,
		BreakSec:     &breakSec,
		ReminderType: &reminderType,
	}); err != nil {
		t.Fatalf("CreateReminder(eye) error = %v", err)
	}

	patches := []config.ReminderPatch{{ID: 999999, Enabled: &enabled}}
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

	name := "eye"
	enabled := true
	intervalSec := 20 * 60
	breakSec := 20
	reminderType := "rest"
	id, err := store.CreateReminder(history.ReminderMutation{
		Name:         &name,
		Enabled:      &enabled,
		IntervalSec:  &intervalSec,
		BreakSec:     &breakSec,
		ReminderType: &reminderType,
	})
	if err != nil {
		t.Fatalf("CreateReminder(eye) error = %v", err)
	}

	zero := 0
	invalidType := "x"
	patches := []config.ReminderPatch{{ID: id, IntervalSec: &zero}}
	if err := applyReminderPatchToHistory(store, patches); err == nil {
		t.Fatalf("expected invalid interval patch to fail")
	}
	patches = []config.ReminderPatch{{ID: id, ReminderType: &invalidType}}
	if err := applyReminderPatchToHistory(store, patches); err == nil {
		t.Fatalf("expected invalid reminder type patch to fail")
	}
}
