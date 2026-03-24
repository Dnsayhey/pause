package history

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func mustCreateReminder(t *testing.T, store *Store, name string) int64 {
	t.Helper()
	enabled := true
	intervalSec := 20 * 60
	breakSec := 20
	reminderType := "rest"
	nameCopy := name
	id, err := store.CreateReminder(ReminderMutation{
		Name:         &nameCopy,
		Enabled:      &enabled,
		IntervalSec:  &intervalSec,
		BreakSec:     &breakSec,
		ReminderType: &reminderType,
	})
	if err != nil {
		t.Fatalf("CreateReminder(%s) error = %v", name, err)
	}
	return id
}

func TestOpenStoreMigratesWithoutSeedingDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	reminders, err := store.ListReminders()
	if err != nil {
		t.Fatalf("ListReminders() error = %v", err)
	}
	if len(reminders) != 0 {
		t.Fatalf("expected no seeded reminders on fresh history db, got %d", len(reminders))
	}
}

func TestStoreRecordsAndAggregatesSessions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	eyeName := "护眼"
	standName := "站立"
	enabled := true
	eyeIntervalSec := 20 * 60
	eyeBreakSec := 20
	standIntervalSec := 60 * 60
	standBreakSec := 5 * 60
	reminderType := "rest"
	eyeID, err := store.CreateReminder(ReminderMutation{
		Name:         &eyeName,
		Enabled:      &enabled,
		IntervalSec:  &eyeIntervalSec,
		BreakSec:     &eyeBreakSec,
		ReminderType: &reminderType,
	})
	if err != nil {
		t.Fatalf("CreateReminder(eye) error = %v", err)
	}
	standID, err := store.CreateReminder(ReminderMutation{
		Name:         &standName,
		Enabled:      &enabled,
		IntervalSec:  &standIntervalSec,
		BreakSec:     &standBreakSec,
		ReminderType: &reminderType,
	})
	if err != nil {
		t.Fatalf("CreateReminder(stand) error = %v", err)
	}

	base := time.Unix(1_700_000_000, 0).UTC()
	s1, err := store.StartBreak(base, "scheduled", 20, []int64{eyeID})
	if err != nil {
		t.Fatalf("StartBreak(s1) error = %v", err)
	}
	if err := store.CompleteBreak(s1, base.Add(20*time.Second), 20); err != nil {
		t.Fatalf("CompleteBreak(s1) error = %v", err)
	}

	s2, err := store.StartBreak(base.Add(1*time.Hour), "manual", 300, []int64{standID})
	if err != nil {
		t.Fatalf("StartBreak(s2) error = %v", err)
	}
	if err := store.SkipBreak(s2, base.Add(1*time.Hour+40*time.Second), 40); err != nil {
		t.Fatalf("SkipBreak(s2) error = %v", err)
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
	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	id := mustCreateReminder(t, store, "eye")
	if _, err := store.db.ExecContext(context.Background(), `UPDATE reminders SET deleted_at = unixepoch() WHERE id = ?`, id); err != nil {
		t.Fatalf("soft delete reminder error = %v", err)
	}

	reminders, err := store.ListReminders()
	if err != nil {
		t.Fatalf("ListReminders() error = %v", err)
	}
	for _, r := range reminders {
		if r.ID == id {
			t.Fatalf("expected soft-deleted reminder to be excluded from list")
		}
	}
}

func TestStartBreakEnforcesSingleRunningSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	base := time.Unix(1_700_000_000, 0).UTC()
	if _, err := store.StartBreak(base, "scheduled", 20, nil); err != nil {
		t.Fatalf("StartBreak(run-1) error = %v", err)
	}
	if _, err := store.StartBreak(base.Add(10*time.Second), "scheduled", 20, nil); err == nil {
		t.Fatalf("expected second running session insert to fail")
	}
}

func TestOpenStoreCancelsDanglingRunningSessions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}

	base := time.Unix(1_700_000_000, 0).UTC()
	sessionID, err := store.StartBreak(base, "manual", 20, nil)
	if err != nil {
		t.Fatalf("StartBreak(dangling) error = %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reopened, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore(reopen) error = %v", err)
	}
	defer reopened.Close()

	var status string
	var endedAt sql.NullInt64
	row := reopened.db.QueryRowContext(context.Background(), `SELECT status, ended_at FROM break_sessions WHERE id = ?`, sessionID)
	if err := row.Scan(&status, &endedAt); err != nil {
		t.Fatalf("scan dangling session error = %v", err)
	}
	if status != "canceled" {
		t.Fatalf("expected dangling running session to be canceled, got %q", status)
	}
	if !endedAt.Valid {
		t.Fatalf("expected dangling running session to have ended_at after cleanup")
	}
}

func TestCreateReminderAndDeleteLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	name := "eye"
	enabled := true
	intervalSec := 20 * 60
	breakSec := 20
	reminderType := "rest"
	id, err := store.CreateReminder(ReminderMutation{
		Name:         &name,
		Enabled:      &enabled,
		IntervalSec:  &intervalSec,
		BreakSec:     &breakSec,
		ReminderType: &reminderType,
	})
	if err != nil {
		t.Fatalf("CreateReminder(eye) error = %v", err)
	}

	if _, err := store.CreateReminder(ReminderMutation{
		Name:         &name,
		Enabled:      &enabled,
		IntervalSec:  &intervalSec,
		BreakSec:     &breakSec,
		ReminderType: &reminderType,
	}); !errors.Is(err, ErrReminderAlreadyExists) {
		t.Fatalf("CreateReminder(existing) error = %v, want %v", err, ErrReminderAlreadyExists)
	}

	if err := store.DeleteReminder(id); err != nil {
		t.Fatalf("DeleteReminder(eye) error = %v", err)
	}
	if err := store.DeleteReminder(id); !errors.Is(err, ErrReminderNotFound) {
		t.Fatalf("DeleteReminder(eye again) error = %v, want %v", err, ErrReminderNotFound)
	}

	restoredID, err := store.CreateReminder(ReminderMutation{
		Name:         &name,
		Enabled:      &enabled,
		IntervalSec:  &intervalSec,
		BreakSec:     &breakSec,
		ReminderType: &reminderType,
	})
	if err != nil {
		t.Fatalf("CreateReminder(restore eye) error = %v", err)
	}
	if restoredID != id {
		t.Fatalf("expected restored reminder id %d, got %d", id, restoredID)
	}
}
