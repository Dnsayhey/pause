package app

import (
	"path/filepath"
	"testing"

	"pause/internal/core/config"
	"pause/internal/core/history"
)

func TestEnsureBuiltInRemindersForFirstInstallSeedsZhNames(t *testing.T) {
	store := openHistoryStoreForSeedTest(t)
	defer store.Close()

	if err := ensureBuiltInRemindersForFirstInstall(store, config.UILanguageZhCN); err != nil {
		t.Fatalf("ensureBuiltInRemindersForFirstInstall() error = %v", err)
	}

	reminders, err := store.ListReminders()
	if err != nil {
		t.Fatalf("ListReminders() error = %v", err)
	}

	eye := requireReminderByID(t, reminders, config.ReminderIDEye)
	if eye.Name != "护眼" {
		t.Fatalf("expected eye name 护眼, got %q", eye.Name)
	}
	if !eye.Enabled {
		t.Fatalf("expected eye enabled=true by default")
	}
	if eye.IntervalSec != 20*60 || eye.BreakSec != 20 {
		t.Fatalf("unexpected eye defaults: interval=%d break=%d", eye.IntervalSec, eye.BreakSec)
	}
	if eye.ReminderType != "rest" {
		t.Fatalf("expected eye type rest, got %q", eye.ReminderType)
	}

	stand := requireReminderByID(t, reminders, config.ReminderIDStand)
	if stand.Name != "站立" {
		t.Fatalf("expected stand name 站立, got %q", stand.Name)
	}
	if stand.Enabled {
		t.Fatalf("expected stand enabled=false by default")
	}
	if stand.IntervalSec != 60*60 || stand.BreakSec != 5*60 {
		t.Fatalf("unexpected stand defaults: interval=%d break=%d", stand.IntervalSec, stand.BreakSec)
	}
	if stand.ReminderType != "rest" {
		t.Fatalf("expected stand type rest, got %q", stand.ReminderType)
	}

	water := requireReminderByID(t, reminders, config.ReminderIDWater)
	if water.Name != "喝水" {
		t.Fatalf("expected water name 喝水, got %q", water.Name)
	}
	if water.Enabled {
		t.Fatalf("expected water enabled=false by default")
	}
	if water.IntervalSec != 45*60 || water.BreakSec != 1 {
		t.Fatalf("unexpected water defaults: interval=%d break=%d", water.IntervalSec, water.BreakSec)
	}
	if water.ReminderType != "notify" {
		t.Fatalf("expected water type notify, got %q", water.ReminderType)
	}
}

func TestEnsureBuiltInRemindersForFirstInstallSeedsEnglishNames(t *testing.T) {
	store := openHistoryStoreForSeedTest(t)
	defer store.Close()

	if err := ensureBuiltInRemindersForFirstInstall(store, config.UILanguageEnUS); err != nil {
		t.Fatalf("ensureBuiltInRemindersForFirstInstall() error = %v", err)
	}

	reminders, err := store.ListReminders()
	if err != nil {
		t.Fatalf("ListReminders() error = %v", err)
	}

	eye := requireReminderByID(t, reminders, config.ReminderIDEye)
	if eye.Name != "Eye" {
		t.Fatalf("expected eye name Eye, got %q", eye.Name)
	}
	stand := requireReminderByID(t, reminders, config.ReminderIDStand)
	if stand.Name != "Stand" {
		t.Fatalf("expected stand name Stand, got %q", stand.Name)
	}
	water := requireReminderByID(t, reminders, config.ReminderIDWater)
	if water.Name != "Hydrate" {
		t.Fatalf("expected water name Hydrate, got %q", water.Name)
	}
}

func TestEnsureBuiltInRemindersForFirstInstallDoesNotOverwriteExistingActive(t *testing.T) {
	store := openHistoryStoreForSeedTest(t)
	defer store.Close()

	customName := "My Eye"
	enabled := false
	intervalSec := 999
	breakSec := 11
	if err := store.CreateReminder(history.ReminderMutation{
		ID:          config.ReminderIDEye,
		Name:        &customName,
		Enabled:     &enabled,
		IntervalSec: &intervalSec,
		BreakSec:    &breakSec,
	}); err != nil {
		t.Fatalf("CreateReminder(eye) error = %v", err)
	}

	if err := ensureBuiltInRemindersForFirstInstall(store, config.UILanguageZhCN); err != nil {
		t.Fatalf("ensureBuiltInRemindersForFirstInstall() error = %v", err)
	}

	reminders, err := store.ListReminders()
	if err != nil {
		t.Fatalf("ListReminders() error = %v", err)
	}
	eye := requireReminderByID(t, reminders, config.ReminderIDEye)
	if eye.Name != customName {
		t.Fatalf("expected existing eye name to remain %q, got %q", customName, eye.Name)
	}
	if eye.Enabled {
		t.Fatalf("expected existing eye enabled=false to remain unchanged")
	}
	if eye.IntervalSec != 999 || eye.BreakSec != 11 {
		t.Fatalf("expected existing eye values unchanged, got interval=%d break=%d", eye.IntervalSec, eye.BreakSec)
	}

	stand := requireReminderByID(t, reminders, config.ReminderIDStand)
	if stand.Name != "站立" {
		t.Fatalf("expected stand to still be seeded, got %q", stand.Name)
	}
	water := requireReminderByID(t, reminders, config.ReminderIDWater)
	if water.Name != "喝水" {
		t.Fatalf("expected water to still be seeded, got %q", water.Name)
	}
}

func openHistoryStoreForSeedTest(t *testing.T) *history.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := history.OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	return store
}

func requireReminderByID(t *testing.T, reminders []history.ReminderDefinition, id string) history.ReminderDefinition {
	t.Helper()
	for _, reminder := range reminders {
		if reminder.ID == id {
			return reminder
		}
	}
	t.Fatalf("expected reminder %q in list", id)
	return history.ReminderDefinition{}
}
