package history

import (
	"math"
	"path/filepath"
	"testing"
	"time"
)

func TestQueryAnalyticsSummary(t *testing.T) {
	store, from, to, _ := prepareAnalyticsTestStore(t)
	defer store.Close()

	summary, err := store.QueryAnalyticsSummary(from, to)
	if err != nil {
		t.Fatalf("QueryAnalyticsSummary() error = %v", err)
	}

	if summary.TotalSessions != 3 {
		t.Fatalf("expected total sessions 3, got %d", summary.TotalSessions)
	}
	if summary.TotalCompleted != 2 {
		t.Fatalf("expected completed 2, got %d", summary.TotalCompleted)
	}
	if summary.TotalSkipped != 1 {
		t.Fatalf("expected skipped 1, got %d", summary.TotalSkipped)
	}
	if summary.TotalCanceled != 0 {
		t.Fatalf("expected canceled 0, got %d", summary.TotalCanceled)
	}
	if summary.TotalActualBreakSec != 38 {
		t.Fatalf("expected total actual break sec 38, got %d", summary.TotalActualBreakSec)
	}
	if !almostEqual(summary.AvgActualBreakSec, 19.0) {
		t.Fatalf("expected avg actual break sec 19.0, got %.4f", summary.AvgActualBreakSec)
	}
	if !almostEqual(summary.CompletionRate, 2.0/3.0) {
		t.Fatalf("expected completion rate 2/3, got %.4f", summary.CompletionRate)
	}
	if !almostEqual(summary.SkipRate, 1.0/3.0) {
		t.Fatalf("expected skip rate 1/3, got %.4f", summary.SkipRate)
	}
	if summary.FromSec != from.UTC().Unix() || summary.ToSec != to.UTC().Unix() {
		t.Fatalf("expected summary range [%d, %d), got [%d, %d)", from.UTC().Unix(), to.UTC().Unix(), summary.FromSec, summary.ToSec)
	}

}

func TestQueryAnalyticsTrendByDay(t *testing.T) {
	store, from, to, fixtures := prepareAnalyticsTestStore(t)
	defer store.Close()

	trend, err := store.QueryAnalyticsTrendByDay(from, to)
	if err != nil {
		t.Fatalf("QueryAnalyticsTrendByDay() error = %v", err)
	}
	if len(trend.Points) != 2 {
		t.Fatalf("expected 2 trend points, got %d", len(trend.Points))
	}

	first := trend.Points[0]
	second := trend.Points[1]
	day1 := fixtures["s1"].startedAt.Local().Format("2006-01-02")
	day2 := fixtures["s3"].startedAt.Local().Format("2006-01-02")

	if first.Day != day1 {
		t.Fatalf("expected first day %q, got %q", day1, first.Day)
	}
	if first.TotalSessions != 2 || first.TotalCompleted != 1 || first.TotalSkipped != 1 {
		t.Fatalf("unexpected day1 counters: %+v", first)
	}
	if !almostEqual(first.CompletionRate, 0.5) || !almostEqual(first.SkipRate, 0.5) {
		t.Fatalf("unexpected day1 rates: completion=%.4f skip=%.4f", first.CompletionRate, first.SkipRate)
	}
	if first.TotalActualBreakSec != 20 || !almostEqual(first.AvgActualBreakSec, 20.0) {
		t.Fatalf("unexpected day1 break stats: total=%d avg=%.4f", first.TotalActualBreakSec, first.AvgActualBreakSec)
	}

	if second.Day != day2 {
		t.Fatalf("expected second day %q, got %q", day2, second.Day)
	}
	if second.TotalSessions != 1 || second.TotalCompleted != 1 || second.TotalSkipped != 0 {
		t.Fatalf("unexpected day2 counters: %+v", second)
	}
	if !almostEqual(second.CompletionRate, 1.0) || !almostEqual(second.SkipRate, 0.0) {
		t.Fatalf("unexpected day2 rates: completion=%.4f skip=%.4f", second.CompletionRate, second.SkipRate)
	}
	if second.TotalActualBreakSec != 18 || !almostEqual(second.AvgActualBreakSec, 18.0) {
		t.Fatalf("unexpected day2 break stats: total=%d avg=%.4f", second.TotalActualBreakSec, second.AvgActualBreakSec)
	}
}

func TestQueryAnalyticsBreakTypeDistribution(t *testing.T) {
	store, from, to, _ := prepareAnalyticsTestStore(t)
	defer store.Close()

	distribution, err := store.QueryAnalyticsBreakTypeDistribution(from, to)
	if err != nil {
		t.Fatalf("QueryAnalyticsBreakTypeDistribution() error = %v", err)
	}
	if len(distribution.Items) < 2 {
		t.Fatalf("expected at least 2 distribution items, got %d", len(distribution.Items))
	}
	if distribution.TotalTriggered != 3 {
		t.Fatalf("expected total triggered 3, got %d", distribution.TotalTriggered)
	}

	byID := map[string]AnalyticsBreakTypeDistributionItem{}
	for _, item := range distribution.Items {
		byID[item.ReminderID] = item
	}

	eye, ok := byID["eye"]
	if !ok {
		t.Fatalf("expected eye reminder in distribution")
	}
	if eye.TriggeredCount != 2 || eye.CompletedCount != 2 || eye.SkippedCount != 0 {
		t.Fatalf("unexpected eye distribution: %+v", eye)
	}
	if !almostEqual(eye.CompletionRate, 1.0) || !almostEqual(eye.SkipRate, 0.0) {
		t.Fatalf("unexpected eye rates: completion=%.4f skip=%.4f", eye.CompletionRate, eye.SkipRate)
	}
	if !almostEqual(eye.TriggeredShare, 2.0/3.0) {
		t.Fatalf("unexpected eye share: %.4f", eye.TriggeredShare)
	}

	stand, ok := byID["stand"]
	if !ok {
		t.Fatalf("expected stand reminder in distribution")
	}
	if stand.TriggeredCount != 1 || stand.CompletedCount != 0 || stand.SkippedCount != 1 {
		t.Fatalf("unexpected stand distribution: %+v", stand)
	}
	if !almostEqual(stand.CompletionRate, 0.0) || !almostEqual(stand.SkipRate, 1.0) {
		t.Fatalf("unexpected stand rates: completion=%.4f skip=%.4f", stand.CompletionRate, stand.SkipRate)
	}
	if !almostEqual(stand.TriggeredShare, 1.0/3.0) {
		t.Fatalf("unexpected stand share: %.4f", stand.TriggeredShare)
	}
}

func TestQueryAnalyticsHourlyHeatmap(t *testing.T) {
	store, from, to, fixtures := prepareAnalyticsTestStore(t)
	defer store.Close()

	heatmap, err := store.QueryAnalyticsHourlyHeatmap(from, to, AnalyticsHeatmapMetricSkipRate)
	if err != nil {
		t.Fatalf("QueryAnalyticsHourlyHeatmap(skip_rate) error = %v", err)
	}
	if heatmap.Metric != AnalyticsHeatmapMetricSkipRate {
		t.Fatalf("expected metric skip_rate, got %q", heatmap.Metric)
	}
	if len(heatmap.Cells) != 3 {
		t.Fatalf("expected 3 heatmap cells, got %d", len(heatmap.Cells))
	}

	type key struct {
		day  string
		hour int
	}
	bySlot := map[key]AnalyticsHeatmapCell{}
	for _, cell := range heatmap.Cells {
		bySlot[key{day: cell.Day, hour: cell.Hour}] = cell
	}

	s1Day := fixtures["s1"].startedAt.Local().Format("2006-01-02")
	s1Hour := fixtures["s1"].startedAt.Local().Hour()
	s2Day := fixtures["s2"].startedAt.Local().Format("2006-01-02")
	s2Hour := fixtures["s2"].startedAt.Local().Hour()
	s3Day := fixtures["s3"].startedAt.Local().Format("2006-01-02")
	s3Hour := fixtures["s3"].startedAt.Local().Hour()

	cell1, ok := bySlot[key{day: s1Day, hour: s1Hour}]
	if !ok {
		t.Fatalf("missing heatmap cell for s1 slot")
	}
	if !almostEqual(cell1.Value, 0.0) {
		t.Fatalf("expected s1 skip rate 0, got %.4f", cell1.Value)
	}

	cell2, ok := bySlot[key{day: s2Day, hour: s2Hour}]
	if !ok {
		t.Fatalf("missing heatmap cell for s2 slot")
	}
	if !almostEqual(cell2.Value, 1.0) {
		t.Fatalf("expected s2 skip rate 1, got %.4f", cell2.Value)
	}

	cell3, ok := bySlot[key{day: s3Day, hour: s3Hour}]
	if !ok {
		t.Fatalf("missing heatmap cell for s3 slot")
	}
	if !almostEqual(cell3.Value, 0.0) {
		t.Fatalf("expected s3 skip rate 0, got %.4f", cell3.Value)
	}
}

func TestQueryAnalyticsHourlyHeatmapRejectsInvalidMetric(t *testing.T) {
	store, from, to, _ := prepareAnalyticsTestStore(t)
	defer store.Close()

	if _, err := store.QueryAnalyticsHourlyHeatmap(from, to, AnalyticsHeatmapMetric("bad_metric")); err == nil {
		t.Fatalf("expected invalid metric to fail")
	}
}

func TestQueryAnalyticsBreakTypeDistributionIncludesDeletedReminderHistory(t *testing.T) {
	store, from, to, _ := prepareAnalyticsTestStore(t)
	defer store.Close()

	if err := store.DeleteReminder("stand"); err != nil {
		t.Fatalf("DeleteReminder(stand) error = %v", err)
	}

	distribution, err := store.QueryAnalyticsBreakTypeDistribution(from, to)
	if err != nil {
		t.Fatalf("QueryAnalyticsBreakTypeDistribution() error = %v", err)
	}

	var stand *AnalyticsBreakTypeDistributionItem
	for idx := range distribution.Items {
		if distribution.Items[idx].ReminderID == "stand" {
			stand = &distribution.Items[idx]
			break
		}
	}
	if stand == nil {
		t.Fatalf("expected deleted reminder history to remain visible")
	}
	if stand.TriggeredCount != 1 {
		t.Fatalf("expected deleted reminder triggered count 1, got %d", stand.TriggeredCount)
	}
}

func TestQueryAnalyticsBreakTypeDistributionUsesSnapshotMetadata(t *testing.T) {
	store, from, to, _ := prepareAnalyticsTestStore(t)
	defer store.Close()

	renamed := "Eye renamed"
	delivery := "notify"
	if err := store.UpdateReminders([]ReminderMutation{{
		ID:           "eye",
		Name:         &renamed,
		ReminderType: &delivery,
	}}); err != nil {
		t.Fatalf("UpdateReminders(eye) error = %v", err)
	}

	distribution, err := store.QueryAnalyticsBreakTypeDistribution(from, to)
	if err != nil {
		t.Fatalf("QueryAnalyticsBreakTypeDistribution() error = %v", err)
	}

	var eye *AnalyticsBreakTypeDistributionItem
	for idx := range distribution.Items {
		if distribution.Items[idx].ReminderID == "eye" {
			eye = &distribution.Items[idx]
			break
		}
	}
	if eye == nil {
		t.Fatalf("expected eye reminder in distribution")
	}
	if eye.ReminderName != "Eye" {
		t.Fatalf("expected snapshot reminder name Eye, got %q", eye.ReminderName)
	}
	if eye.ReminderType != "rest" {
		t.Fatalf("expected snapshot reminder type rest, got %q", eye.ReminderType)
	}
}

type sessionFixture struct {
	startedAt time.Time
}

func prepareAnalyticsTestStore(t *testing.T) (*Store, time.Time, time.Time, map[string]sessionFixture) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "history.db")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}

	if err := store.SyncReminders([]ReminderDefinition{
		{ID: "eye", Name: "Eye", Enabled: true, IntervalSec: 1200, BreakSec: 20, ReminderType: "rest"},
		{ID: "stand", Name: "Stand", Enabled: true, IntervalSec: 3600, BreakSec: 300, ReminderType: "rest"},
	}); err != nil {
		t.Fatalf("SyncReminders() error = %v", err)
	}

	base := time.Date(2025, 1, 3, 9, 0, 0, 0, time.UTC)

	s1 := base
	if err := store.StartBreak("s1", s1, "scheduled", 20, []string{"eye"}); err != nil {
		t.Fatalf("StartBreak(s1) error = %v", err)
	}
	if err := store.CompleteBreak("s1", s1.Add(20*time.Second), 20); err != nil {
		t.Fatalf("CompleteBreak(s1) error = %v", err)
	}

	s2 := base.Add(1 * time.Hour)
	if err := store.StartBreak("s2", s2, "manual", 300, []string{"stand"}); err != nil {
		t.Fatalf("StartBreak(s2) error = %v", err)
	}
	if err := store.SkipBreak("s2", s2.Add(40*time.Second), 40); err != nil {
		t.Fatalf("SkipBreak(s2) error = %v", err)
	}

	s3 := base.Add(26 * time.Hour)
	if err := store.StartBreak("s3", s3, "scheduled", 20, []string{"eye"}); err != nil {
		t.Fatalf("StartBreak(s3) error = %v", err)
	}
	if err := store.CompleteBreak("s3", s3.Add(18*time.Second), 18); err != nil {
		t.Fatalf("CompleteBreak(s3) error = %v", err)
	}

	from := base.Add(-time.Hour)
	to := s3.Add(2 * time.Hour)
	fixtures := map[string]sessionFixture{
		"s1": {startedAt: s1},
		"s2": {startedAt: s2},
		"s3": {startedAt: s3},
	}
	return store, from, to, fixtures
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}
