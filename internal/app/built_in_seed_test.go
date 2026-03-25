package app

import (
	"context"
	"path/filepath"
	"testing"

	"pause/internal/backend/bootstrap"
	"pause/internal/backend/domain/settings"
	"pause/internal/backend/storage/historydb"
)

func openSeedStore(t *testing.T) *historydb.Store {
	t.Helper()
	store, err := historydb.OpenStore(context.Background(), filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatalf("OpenStore() err=%v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func openSeedReminderService(t *testing.T, store *historydb.Store) reminderService {
	t.Helper()
	container, err := bootstrap.NewContainer(store)
	if err != nil {
		t.Fatalf("NewContainer() err=%v", err)
	}
	return container.ReminderService
}

func findReminderByName(t *testing.T, reminders []historydb.Reminder, name string) historydb.Reminder {
	t.Helper()
	for _, r := range reminders {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("missing reminder name=%q", name)
	return historydb.Reminder{}
}

func TestEnsureBuiltInReminders_ZhCN(t *testing.T) {
	store := openSeedStore(t)
	svc := openSeedReminderService(t, store)
	if err := ensureBuiltInRemindersForFirstInstall(context.Background(), svc, settings.UILanguageZhCN); err != nil {
		t.Fatalf("ensureBuiltInRemindersForFirstInstall() err=%v", err)
	}
	items, err := store.ListReminders(context.Background())
	if err != nil {
		t.Fatalf("ListReminders() err=%v", err)
	}
	findReminderByName(t, items, "护眼")
	findReminderByName(t, items, "站立")
	findReminderByName(t, items, "喝水")
}

func TestEnsureBuiltInReminders_EnUS(t *testing.T) {
	store := openSeedStore(t)
	svc := openSeedReminderService(t, store)
	if err := ensureBuiltInRemindersForFirstInstall(context.Background(), svc, settings.UILanguageEnUS); err != nil {
		t.Fatalf("ensureBuiltInRemindersForFirstInstall() err=%v", err)
	}
	items, err := store.ListReminders(context.Background())
	if err != nil {
		t.Fatalf("ListReminders() err=%v", err)
	}
	findReminderByName(t, items, "Eye")
	findReminderByName(t, items, "Stand")
	findReminderByName(t, items, "Hydrate")
}
