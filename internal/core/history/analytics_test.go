package history

import (
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
	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}

	eyeName := "Eye"
	standName := "Stand"
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

	s2, err := store.StartBreak(base.Add(2*time.Hour), "manual", 300, []int64{standID})
	if err != nil {
		t.Fatalf("StartBreak(s2) error = %v", err)
	}
	if err := store.SkipBreak(s2, base.Add(2*time.Hour+40*time.Second), 40); err != nil {
		t.Fatalf("SkipBreak(s2) error = %v", err)
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

func TestQueryAnalyticsHourlyHeatmapRejectsInvalidMetric(t *testing.T) {
	store, fixture := prepareAnalyticsFixture(t)
	defer store.Close()

	if _, err := store.QueryAnalyticsHourlyHeatmap(fixture.from, fixture.to, AnalyticsHeatmapMetric("bad_metric")); err == nil {
		t.Fatalf("expected invalid metric to fail")
	}
}
