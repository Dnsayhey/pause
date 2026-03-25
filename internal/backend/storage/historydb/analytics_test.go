package historydb

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

type analyticsFixture struct {
	eyeID   int64
	standID int64
	from    time.Time
	to      time.Time
}

func prepareAnalyticsFixture(t *testing.T) (*Store, analyticsFixture) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := OpenStore(context.Background(), path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}

	eyeID, err := store.CreateReminder(context.Background(), Reminder{
		Name:         "Eye",
		Enabled:      true,
		IntervalSec:  20 * 60,
		BreakSec:     20,
		ReminderType: "rest",
	})
	if err != nil {
		t.Fatalf("CreateReminder(eye) error = %v", err)
	}
	standID, err := store.CreateReminder(context.Background(), Reminder{
		Name:         "Stand",
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
		base.Add(2*time.Hour),
		base.Add(2*time.Hour+40*time.Second),
		"manual",
		300,
		40,
		true,
		[]int64{standID},
	); err != nil {
		t.Fatalf("RecordBreak(s2) error = %v", err)
	}

	return store, analyticsFixture{
		eyeID:   eyeID,
		standID: standID,
		from:    base.Add(-time.Hour),
		to:      base.Add(24 * time.Hour),
	}
}

func TestQueryAnalyticsSummary(t *testing.T) {
	store, fixture := prepareAnalyticsFixture(t)
	defer store.Close()

	summary, err := store.QueryAnalyticsSummary(fixture.from, fixture.to)
	if err != nil {
		t.Fatalf("QueryAnalyticsSummary() error = %v", err)
	}
	if summary.TotalSessions != 2 {
		t.Fatalf("expected total sessions 2, got %d", summary.TotalSessions)
	}
	if summary.TotalCompleted != 1 {
		t.Fatalf("expected completed 1, got %d", summary.TotalCompleted)
	}
	if summary.TotalSkipped != 1 {
		t.Fatalf("expected skipped 1, got %d", summary.TotalSkipped)
	}
}

func TestQueryAnalyticsBreakTypeDistribution(t *testing.T) {
	store, fixture := prepareAnalyticsFixture(t)
	defer store.Close()

	distribution, err := store.QueryAnalyticsBreakTypeDistribution(fixture.from, fixture.to)
	if err != nil {
		t.Fatalf("QueryAnalyticsBreakTypeDistribution() error = %v", err)
	}
	if distribution.TotalTriggered != 2 {
		t.Fatalf("expected total triggered 2, got %d", distribution.TotalTriggered)
	}

	byID := map[int64]AnalyticsBreakTypeDistributionItem{}
	for _, item := range distribution.Items {
		byID[item.ReminderID] = item
	}
	if _, ok := byID[fixture.eyeID]; !ok {
		t.Fatalf("expected eye reminder distribution item")
	}
	if _, ok := byID[fixture.standID]; !ok {
		t.Fatalf("expected stand reminder distribution item")
	}
}
