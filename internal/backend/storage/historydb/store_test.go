package historydb

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	reminderdomain "pause/internal/backend/domain/reminder"
)

func openStoreForTest(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := OpenStore(context.Background(), path)
	if err != nil {
		t.Fatalf("OpenStore() err=%v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func createReminder(t *testing.T, store *Store, name string) int64 {
	t.Helper()
	id, err := store.CreateReminder(context.Background(), Reminder{Name: name, Enabled: true, IntervalSec: 20 * 60, BreakSec: 20, ReminderType: "rest"})
	if err != nil {
		t.Fatalf("CreateReminder(%s) err=%v", name, err)
	}
	return id
}

func TestHistoryStore_FreshDBHasNoSeedData(t *testing.T) {
	store := openStoreForTest(t)
	items, err := store.ListReminders(context.Background())
	if err != nil {
		t.Fatalf("ListReminders() err=%v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty reminders on fresh DB, got=%d", len(items))
	}
}

func TestHistoryStore_CreateDeleteRestoreSameName(t *testing.T) {
	store := openStoreForTest(t)
	id := createReminder(t, store, "eye")

	if _, err := store.CreateReminder(context.Background(), Reminder{Name: "eye", Enabled: true, IntervalSec: 20 * 60, BreakSec: 20, ReminderType: "rest"}); !errors.Is(err, ErrReminderAlreadyExists) {
		t.Fatalf("expected already exists, got=%v", err)
	}
	if err := store.DeleteReminder(context.Background(), id); err != nil {
		t.Fatalf("DeleteReminder() err=%v", err)
	}
	if err := store.DeleteReminder(context.Background(), id); !errors.Is(err, ErrReminderNotFound) {
		t.Fatalf("expected not found on second delete, got=%v", err)
	}

	restored, err := store.CreateReminder(context.Background(), Reminder{Name: "eye", Enabled: true, IntervalSec: 20 * 60, BreakSec: 20, ReminderType: "rest"})
	if err != nil {
		t.Fatalf("CreateReminder(restore) err=%v", err)
	}
	if restored != id {
		t.Fatalf("expected restored id reuse got=%d want=%d", restored, id)
	}
}

func TestHistoryStore_RecordBreakRejectsDuplicateReminderIDs(t *testing.T) {
	store := openStoreForTest(t)
	base := time.Unix(1_700_000_000, 0).UTC()
	id := createReminder(t, store, "eye")

	err := store.RecordBreak(context.Background(), base, base.Add(20*time.Second), "scheduled", 20, 20, false, []int64{id, id})
	if err == nil {
		t.Fatalf("expected duplicate reminder ids error")
	}
}

func TestHistoryStore_UpdateReminderValidation(t *testing.T) {
	store := openStoreForTest(t)
	if err := store.UpdateReminder(context.Background(), 0, ReminderPatch{}); err == nil {
		t.Fatalf("expected invalid reminder id error")
	}
	if err := store.UpdateReminder(context.Background(), 999, ReminderPatch{}); !errors.Is(err, ErrReminderNotFound) {
		t.Fatalf("expected reminder not found, got=%v", err)
	}
}

func TestHistoryStore_CreateReminderValidationUsesWrappedError(t *testing.T) {
	store := openStoreForTest(t)
	_, err := store.CreateReminder(context.Background(), Reminder{
		Name:         "Eye",
		Enabled:      true,
		IntervalSec:  0,
		BreakSec:     20,
		ReminderType: "rest",
	})
	if !errors.Is(err, reminderdomain.ErrIntervalRange) {
		t.Fatalf("expected interval validation error, got=%v", err)
	}
	if !strings.Contains(err.Error(), "history create reminder validate failed") {
		t.Fatalf("expected wrapped validation message, got=%v", err)
	}
}

func TestHistoryStore_DeleteReminderPreservesNotFoundSentinel(t *testing.T) {
	store := openStoreForTest(t)
	if err := store.DeleteReminder(context.Background(), 999); !errors.Is(err, ErrReminderNotFound) {
		t.Fatalf("expected not found, got=%v", err)
	}
}
