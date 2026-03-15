package history

import (
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
