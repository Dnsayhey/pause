package app

import (
	"context"
	"path/filepath"
	"testing"

	"pause/internal/core/config"
	"pause/internal/core/history"
)

func TestEnsureBuiltInRemindersForFirstInstallSeedsZhNames(t *testing.T) {
	store := openHistoryStoreForSeedTest(t)
	defer store.Close()

	if err := ensureBuiltInRemindersForFirstInstall(context.Background(), store, config.UILanguageZhCN); err != nil {
		t.Fatalf("ensureBuiltInRemindersForFirstInstall() error = %v", err)
	}

	reminders, err := store.ListReminders(context.Background())
	if err != nil {
		t.Fatalf("ListReminders() error = %v", err)
	}

	eye := requireReminderByName(t, reminders, "护眼")
	if !eye.Enabled {
		t.Fatalf("expected eye enabled=true by default")
	}
	if eye.IntervalSec != 20*60 || eye.BreakSec != 20 {
		t.Fatalf("unexpected eye defaults: interval=%d break=%d", eye.IntervalSec, eye.BreakSec)
	}
	if eye.ReminderType != "rest" {
		t.Fatalf("expected eye type rest, got %q", eye.ReminderType)
	}

	stand := requireReminderByName(t, reminders, "站立")
	if stand.Enabled {
		t.Fatalf("expected stand enabled=false by default")
	}
	if stand.IntervalSec != 60*60 || stand.BreakSec != 5*60 {
		t.Fatalf("unexpected stand defaults: interval=%d break=%d", stand.IntervalSec, stand.BreakSec)
	}
	if stand.ReminderType != "rest" {
		t.Fatalf("expected stand type rest, got %q", stand.ReminderType)
	}

	water := requireReminderByName(t, reminders, "喝水")
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

	if err := ensureBuiltInRemindersForFirstInstall(context.Background(), store, config.UILanguageEnUS); err != nil {
		t.Fatalf("ensureBuiltInRemindersForFirstInstall() error = %v", err)
	}

	reminders, err := store.ListReminders(context.Background())
	if err != nil {
		t.Fatalf("ListReminders() error = %v", err)
	}

	requireReminderByName(t, reminders, "Eye")
	requireReminderByName(t, reminders, "Stand")
	requireReminderByName(t, reminders, "Hydrate")
}

func TestEnsureBuiltInRemindersForFirstInstallDoesNotOverwriteExistingActive(t *testing.T) {
	store := openHistoryStoreForSeedTest(t)
	defer store.Close()

	customName := "护眼"
	enabled := false
	intervalSec := 999
	breakSec := 11
	reminderType := "rest"
	if _, err := store.CreateReminder(context.Background(), history.Reminder{
		Name:         customName,
		Enabled:      enabled,
		IntervalSec:  intervalSec,
		BreakSec:     breakSec,
		ReminderType: reminderType,
	}); err != nil {
		t.Fatalf("CreateReminder(eye) error = %v", err)
	}

	if err := ensureBuiltInRemindersForFirstInstall(context.Background(), store, config.UILanguageZhCN); err != nil {
		t.Fatalf("ensureBuiltInRemindersForFirstInstall() error = %v", err)
	}

	reminders, err := store.ListReminders(context.Background())
	if err != nil {
		t.Fatalf("ListReminders() error = %v", err)
	}
	eye := requireReminderByName(t, reminders, "护眼")
	if eye.Enabled {
		t.Fatalf("expected existing eye enabled=false to remain unchanged")
	}
	if eye.IntervalSec != 999 || eye.BreakSec != 11 {
		t.Fatalf("expected existing eye values unchanged, got interval=%d break=%d", eye.IntervalSec, eye.BreakSec)
	}

	requireReminderByName(t, reminders, "站立")
	requireReminderByName(t, reminders, "喝水")
}

func openHistoryStoreForSeedTest(t *testing.T) *history.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := history.OpenStore(context.Background(), path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	return store
}

func requireReminderByName(t *testing.T, reminders []history.Reminder, name string) history.Reminder {
	t.Helper()
	for _, reminder := range reminders {
		if reminder.Name == name {
			return reminder
		}
	}
	t.Fatalf("expected reminder %q in list", name)
	return history.Reminder{}
}
