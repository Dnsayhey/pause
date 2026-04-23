package historydb

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func analyticsFixtureStore(t *testing.T) *Store {
	t.Helper()
	store, err := OpenStore(context.Background(), filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatalf("OpenStore() err=%v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	eyeID, err := store.CreateReminder(context.Background(), Reminder{Name: "Eye", Enabled: true, IntervalSec: 20 * 60, BreakSec: 20, ReminderType: "rest"})
	if err != nil {
		t.Fatalf("CreateReminder(Eye) err=%v", err)
	}
	standID, err := store.CreateReminder(context.Background(), Reminder{Name: "Stand", Enabled: true, IntervalSec: 60 * 60, BreakSec: 300, ReminderType: "rest"})
	if err != nil {
		t.Fatalf("CreateReminder(Stand) err=%v", err)
	}

	base := time.Unix(1_700_000_000, 0).UTC()
	if err := store.RecordBreak(context.Background(), base, base.Add(20*time.Second), "scheduled", 20, 20, false, []int64{eyeID}); err != nil {
		t.Fatalf("RecordBreak(completed) err=%v", err)
	}
	if err := store.RecordBreak(context.Background(), base.Add(time.Hour), base.Add(time.Hour+40*time.Second), "manual", 300, 40, true, []int64{standID}); err != nil {
		t.Fatalf("RecordBreak(skipped) err=%v", err)
	}
	return store
}

func TestAnalytics_QuerySummary(t *testing.T) {
	store := analyticsFixtureStore(t)
	base := time.Unix(1_700_000_000, 0).UTC()

	summary, err := store.QueryAnalyticsSummary(context.Background(), base.Add(-time.Hour), base.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("QueryAnalyticsSummary() err=%v", err)
	}
	if summary.TotalSessions != 2 || summary.TotalCompleted != 1 || summary.TotalSkipped != 1 {
		t.Fatalf("summary counters mismatch: %+v", summary)
	}
	if summary.TotalActualBreakSec != 20 {
		t.Fatalf("unexpected total actual break sec=%d", summary.TotalActualBreakSec)
	}
}

func TestAnalytics_QueryBreakTypeDistribution(t *testing.T) {
	store := analyticsFixtureStore(t)
	base := time.Unix(1_700_000_000, 0).UTC()

	dist, err := store.QueryAnalyticsBreakTypeDistribution(context.Background(), base.Add(-time.Hour), base.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("QueryAnalyticsBreakTypeDistribution() err=%v", err)
	}
	if dist.TotalTriggered != 2 {
		t.Fatalf("total triggered mismatch: got=%d want=2", dist.TotalTriggered)
	}
	if len(dist.Items) == 0 {
		t.Fatalf("expected distribution items")
	}
}

func TestAnalytics_QuerySummaryHonorsCanceledContext(t *testing.T) {
	store := analyticsFixtureStore(t)
	base := time.Unix(1_700_000_000, 0).UTC()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.QueryAnalyticsSummary(ctx, base.Add(-time.Hour), base.Add(24*time.Hour))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("QueryAnalyticsSummary() err=%v want=%v", err, context.Canceled)
	}
}
