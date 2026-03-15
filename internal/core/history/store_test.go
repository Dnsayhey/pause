package history

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenStoreMigratesAndSeedsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	stats, err := store.QueryWeeklyStats(time.Unix(0, 0), time.Unix(4_102_444_800, 0))
	if err != nil {
		t.Fatalf("QueryWeeklyStats() error = %v", err)
	}
	if len(stats.Reminders) < 2 {
		t.Fatalf("expected seeded reminders, got %d", len(stats.Reminders))
	}
}

func TestStoreRecordsAndAggregatesSessions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	if err := store.SyncReminders([]ReminderDefinition{
		{ID: "eye", Name: "护眼", Enabled: true, IntervalSec: 1200, BreakSec: 20, DeliveryType: "overlay"},
		{ID: "stand", Name: "站立", Enabled: true, IntervalSec: 3600, BreakSec: 300, DeliveryType: "overlay"},
	}); err != nil {
		t.Fatalf("SyncReminders() error = %v", err)
	}

	base := time.Unix(1_700_000_000, 0).UTC()
	if err := store.StartBreak("s1", base, "scheduled", 20, []string{"eye"}); err != nil {
		t.Fatalf("StartBreak(s1) error = %v", err)
	}
	if err := store.CompleteBreak("s1", base.Add(20*time.Second), 20); err != nil {
		t.Fatalf("CompleteBreak(s1) error = %v", err)
	}

	if err := store.StartBreak("s2", base.Add(1*time.Hour), "manual", 300, []string{"stand"}); err != nil {
		t.Fatalf("StartBreak(s2) error = %v", err)
	}
	if err := store.SkipBreak("s2", base.Add(1*time.Hour+40*time.Second), 40); err != nil {
		t.Fatalf("SkipBreak(s2) error = %v", err)
	}

	stats, err := store.QueryWeeklyStats(base.Add(-time.Hour), base.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("QueryWeeklyStats() error = %v", err)
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

	if _, err := store.db.ExecContext(context.Background(), `UPDATE reminders SET deleted_at = unixepoch() WHERE id = 'eye'`); err != nil {
		t.Fatalf("soft delete reminder error = %v", err)
	}

	reminders, err := store.ListReminders()
	if err != nil {
		t.Fatalf("ListReminders() error = %v", err)
	}
	for _, r := range reminders {
		if r.ID == "eye" {
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
	if err := store.StartBreak("run-1", base, "scheduled", 20, nil); err != nil {
		t.Fatalf("StartBreak(run-1) error = %v", err)
	}
	if err := store.StartBreak("run-2", base.Add(10*time.Second), "scheduled", 20, nil); err == nil {
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
	if err := store.StartBreak("dangling", base, "manual", 20, nil); err != nil {
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
	row := reopened.db.QueryRowContext(context.Background(), `SELECT status, ended_at FROM break_sessions WHERE id = ?`, "dangling")
	if err := row.Scan(&status, &endedAt); err != nil {
		t.Fatalf("scan dangling session error = %v", err)
	}
	if status != "canceled" {
		t.Fatalf("expected dangling running session to be canceled, got %q", status)
	}
	if !endedAt.Valid {
		t.Fatalf("expected dangling running session to have ended_at after cleanup")
	}

	// After cleanup there should be no running rows left.
	var runningCount int
	if err := reopened.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM break_sessions WHERE status = 'running'`).Scan(&runningCount); err != nil {
		t.Fatalf("count running sessions error = %v", err)
	}
	if runningCount != 0 {
		t.Fatalf("expected zero running sessions after reopen cleanup, got %d", runningCount)
	}
}
