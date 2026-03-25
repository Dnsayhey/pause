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

func (s *analyticsRepoStub) QueryWeeklyStats(_ context.Context, from time.Time, to time.Time) (analyticsdomain.WeeklyStats, error) {
	s.lastFrom = from
	s.lastTo = to
	return analyticsdomain.WeeklyStats{}, nil
}

func (s *analyticsRepoStub) QuerySummary(_ context.Context, from time.Time, to time.Time) (analyticsdomain.Summary, error) {
	s.lastFrom = from
	s.lastTo = to
	return analyticsdomain.Summary{FromSec: from.Unix(), ToSec: to.Unix()}, nil
}

func (s *analyticsRepoStub) QueryTrendByDay(_ context.Context, from time.Time, to time.Time) (analyticsdomain.Trend, error) {
	s.lastFrom = from
	s.lastTo = to
	return analyticsdomain.Trend{}, nil
}

func (s *analyticsRepoStub) QueryBreakTypeDistribution(_ context.Context, from time.Time, to time.Time) (analyticsdomain.BreakTypeDistribution, error) {
	s.lastFrom = from
	s.lastTo = to
	return analyticsdomain.BreakTypeDistribution{}, nil
}

func TestServiceGetSummaryRejectsInvalidRange(t *testing.T) {
	repo := &analyticsRepoStub{}
	svc, err := NewService(repo)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, err := svc.GetSummary(context.Background(), 200, 100); err == nil {
		t.Fatalf("expected invalid time range error")
	}
}

func TestServiceGetSummaryUsesCurrentWeekWhenRangeMissing(t *testing.T) {
	repo := &analyticsRepoStub{}
	svc, err := NewService(repo)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	now := time.Date(2026, 3, 25, 10, 30, 0, 0, time.Local)
	svc.now = func() time.Time { return now }
	wantFrom, wantTo := currentWeekRange(now)

	summary, err := svc.GetSummary(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("GetSummary() error = %v", err)
	}
	if summary.FromSec != wantFrom.Unix() || summary.ToSec != wantTo.Unix() {
		t.Fatalf("expected range [%d, %d), got [%d, %d)", wantFrom.Unix(), wantTo.Unix(), summary.FromSec, summary.ToSec)
	}
}
