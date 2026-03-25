package analytics

import (
	"context"
	"testing"
	"time"

	analyticsdomain "pause/internal/backend/domain/analytics"
)

type analyticsRepoStub struct {
	lastFrom time.Time
	lastTo   time.Time
}

func (s *analyticsRepoStub) QueryWeeklyStats(_ context.Context, from, to time.Time) (analyticsdomain.WeeklyStats, error) {
	s.lastFrom, s.lastTo = from, to
	return analyticsdomain.WeeklyStats{}, nil
}

func (s *analyticsRepoStub) QuerySummary(_ context.Context, from, to time.Time) (analyticsdomain.Summary, error) {
	s.lastFrom, s.lastTo = from, to
	return analyticsdomain.Summary{FromSec: from.Unix(), ToSec: to.Unix()}, nil
}

func (s *analyticsRepoStub) QueryTrendByDay(_ context.Context, from, to time.Time) (analyticsdomain.Trend, error) {
	s.lastFrom, s.lastTo = from, to
	return analyticsdomain.Trend{}, nil
}

func (s *analyticsRepoStub) QueryBreakTypeDistribution(_ context.Context, from, to time.Time) (analyticsdomain.BreakTypeDistribution, error) {
	s.lastFrom, s.lastTo = from, to
	return analyticsdomain.BreakTypeDistribution{}, nil
}

func TestAnalyticsService_InvalidRange(t *testing.T) {
	svc, err := NewService(&analyticsRepoStub{})
	if err != nil {
		t.Fatalf("NewService() err=%v", err)
	}
	if _, err := svc.GetSummary(context.Background(), 20, 10); err == nil {
		t.Fatalf("expected invalid range error")
	}
}

func TestAnalyticsService_DefaultRangeUsesCurrentWeek(t *testing.T) {
	repo := &analyticsRepoStub{}
	svc, err := NewService(repo)
	if err != nil {
		t.Fatalf("NewService() err=%v", err)
	}

	now := time.Date(2026, 3, 25, 9, 0, 0, 0, time.Local)
	svc.now = func() time.Time { return now }
	wantFrom, wantTo := currentWeekRange(now)

	summary, err := svc.GetSummary(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("GetSummary() err=%v", err)
	}
	if summary.FromSec != wantFrom.Unix() || summary.ToSec != wantTo.Unix() {
		t.Fatalf("summary range mismatch: got=[%d,%d) want=[%d,%d)", summary.FromSec, summary.ToSec, wantFrom.Unix(), wantTo.Unix())
	}
}
