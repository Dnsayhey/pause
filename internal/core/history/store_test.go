package history

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func mustCreateReminder(t *testing.T, store *HistoryStore, name string) int64 {
	t.Helper()
	id, err := store.CreateReminder(context.Background(), Reminder{
		Name:         name,
		Enabled:      true,
		IntervalSec:  20 * 60,
		BreakSec:     20,
		ReminderType: "rest",
	})
	if err != nil {
		t.Fatalf("CreateReminder(%s) error = %v", name, err)
	}
	return id
}

func TestOpenStoreMigratesWithoutSeedingDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := OpenHistoryStore(context.Background(), path)
	if err != nil {
		t.Fatalf("OpenHistoryStore() error = %v", err)
	}
	defer store.Close()

	reminders, err := store.ListReminders(context.Background())
	if err != nil {
		t.Fatalf("ListReminders() error = %v", err)
	}
	if len(reminders) != 0 {
		t.Fatalf("expected no seeded reminders on fresh history db, got %d", len(reminders))
	}
}

func TestStoreRecordsAndAggregatesSessions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := OpenHistoryStore(context.Background(), path)
	if err != nil {
		t.Fatalf("OpenHistoryStore() error = %v", err)
	}
	defer store.Close()

	eyeID, err := store.CreateReminder(context.Background(), Reminder{
		Name:         "护眼",
		Enabled:      true,
		IntervalSec:  20 * 60,
		BreakSec:     20,
		ReminderType: "rest",
	})
	if err != nil {
		t.Fatalf("CreateReminder(eye) error = %v", err)
	}
	standID, err := store.CreateReminder(context.Background(), Reminder{
		Name:         "站立",
		Enabled:      true,
		IntervalSec:  60 * 60,
		BreakSec:     5 * 60,
		ReminderType: "rest",
	})
	if err != nil {
		t.Fatalf("CreateReminder(stand) error = %v", err)
	}

	base := time.Unix(1_700_000_000, 0).UTC()
	if err := store.RecordBreak(context.Background(), base, base.Add(20*time.Second), "scheduled", 20, 20, false, []int64{eyeID}); err != nil {
		t.Fatalf("RecordBreak(s1) error = %v", err)
	}

	if err := store.RecordBreak(
		context.Background(),
		base.Add(1*time.Hour),
		base.Add(1*time.Hour+40*time.Second),
		"manual",
		300,
		40,
		true,
		[]int64{standID},
	); err != nil {
		t.Fatalf("RecordBreak(s2) error = %v", err)
	}

	stats, err := store.QueryAnalyticsWeeklyStats(base.Add(-time.Hour), base.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("QueryAnalyticsWeeklyStats() error = %v", err)
	}
	if stats.Summary.TotalSessions != 2 {
		t.Fatalf("expected total sessions 2, got %d", stats.Summary.TotalSessions)
	}
	if stats.Summary.TotalCompleted != 1 {
		t.Fatalf("expected total completed 1, got %d", stats.Summary.TotalCompleted)
	}
	if stats.Summary.TotalSkipped != 1 {
		t.Fatalf("expected total skipped 1, got %d", stats.Summary.TotalSkipped)
	}
	if stats.Summary.TotalActualBreakSec != 20 {
		t.Fatalf("expected total completed actual sec 20, got %d", stats.Summary.TotalActualBreakSec)
	}
}

func TestListRemindersSkipsSoftDeletedRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := OpenHistoryStore(context.Background(), path)
	if err != nil {
		t.Fatalf("OpenHistoryStore() error = %v", err)
	}
	defer store.Close()

	id := mustCreateReminder(t, store, "eye")
	if _, err := store.db.ExecContext(context.Background(), `UPDATE reminders SET deleted_at = unixepoch() WHERE id = ?`, id); err != nil {
		t.Fatalf("soft delete reminder error = %v", err)
	}

	reminders, err := store.ListReminders(context.Background())
	if err != nil {
		t.Fatalf("ListReminders() error = %v", err)
	}
	for _, r := range reminders {
		if r.ID == id {
			t.Fatalf("expected soft-deleted reminder to be excluded from list")
		}
	}
}

func TestRecordBreakRejectsDuplicateReminderIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := OpenHistoryStore(context.Background(), path)
	if err != nil {
		t.Fatalf("OpenHistoryStore() error = %v", err)
	}
	defer store.Close()

	base := time.Unix(1_700_000_000, 0).UTC()
	id := mustCreateReminder(t, store, "eye")
	err = store.RecordBreak(
		context.Background(),
		base,
		base.Add(20*time.Second),
		"scheduled",
		20,
		20,
		false,
		[]int64{id, id},
	)
	if err == nil {
		t.Fatalf("expected duplicate reminder ids to fail")
	}
}

func TestCreateReminderAndDeleteLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := OpenHistoryStore(context.Background(), path)
	if err != nil {
		t.Fatalf("OpenHistoryStore() error = %v", err)
	}
	defer store.Close()

	id, err := store.CreateReminder(context.Background(), Reminder{
		Name:         "eye",
		Enabled:      true,
		IntervalSec:  20 * 60,
		BreakSec:     20,
		ReminderType: "rest",
	})
	if err != nil {
		t.Fatalf("CreateReminder(eye) error = %v", err)
	}

	if _, err := store.CreateReminder(context.Background(), Reminder{
		Name:         "eye",
		Enabled:      true,
		IntervalSec:  20 * 60,
		BreakSec:     20,
		ReminderType: "rest",
	}); !errors.Is(err, ErrReminderAlreadyExists) {
		t.Fatalf("CreateReminder(existing) error = %v, want %v", err, ErrReminderAlreadyExists)
	}

	if err := store.DeleteReminder(context.Background(), id); err != nil {
		t.Fatalf("DeleteReminder(eye) error = %v", err)
	}
	if err := store.DeleteReminder(context.Background(), id); !errors.Is(err, ErrReminderNotFound) {
		t.Fatalf("DeleteReminder(eye again) error = %v, want %v", err, ErrReminderNotFound)
	}

	restoredID, err := store.CreateReminder(context.Background(), Reminder{
		Name:         "eye",
		Enabled:      true,
		IntervalSec:  20 * 60,
		BreakSec:     20,
		ReminderType: "rest",
	})
	if err != nil {
		t.Fatalf("CreateReminder(restore eye) error = %v", err)
	}
	if restoredID != id {
		t.Fatalf("expected restored reminder id %d, got %d", id, restoredID)
	}
}

func TestUpdateReminderRejectsInvalidID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := OpenHistoryStore(context.Background(), path)
	if err != nil {
		t.Fatalf("OpenHistoryStore() error = %v", err)
	}
	defer store.Close()

	err = store.UpdateReminder(context.Background(), 0, ReminderPatch{})
	if err == nil {
		t.Fatalf("expected invalid reminder id to fail")
	}
	if err.Error() != "reminder id is required" {
		t.Fatalf("unexpected error for invalid id: %v", err)
	}
}

func TestUpdateReminderRejectsUnknownReminderID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := OpenHistoryStore(context.Background(), path)
	if err != nil {
		t.Fatalf("OpenHistoryStore() error = %v", err)
	}
	defer store.Close()

	enabled := true
	err = store.UpdateReminder(context.Background(), 999999, ReminderPatch{Enabled: &enabled})
	if !errors.Is(err, ErrReminderNotFound) {
		t.Fatalf("UpdateReminder(unknown) error = %v, want %v", err, ErrReminderNotFound)
	}
}

func TestUpdateReminderRejectsSoftDeletedReminder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := OpenHistoryStore(context.Background(), path)
	if err != nil {
		t.Fatalf("OpenHistoryStore() error = %v", err)
	}
	defer store.Close()

	id := mustCreateReminder(t, store, "eye")
	if err := store.DeleteReminder(context.Background(), id); err != nil {
		t.Fatalf("DeleteReminder(eye) error = %v", err)
	}

	enabled := true
	err = store.UpdateReminder(context.Background(), id, ReminderPatch{Enabled: &enabled})
	if !errors.Is(err, ErrReminderNotFound) {
		t.Fatalf("UpdateReminder(soft-deleted) error = %v, want %v", err, ErrReminderNotFound)
	}
}
