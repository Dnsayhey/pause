package analytics

import (
	"context"
	"errors"
	"time"

	analyticsdomain "pause/internal/backend/domain/analytics"
	"pause/internal/backend/ports"
)

type Service struct {
	repo ports.AnalyticsRepository
	now  func() time.Time
}

func NewService(repo ports.AnalyticsRepository) (*Service, error) {
	if repo == nil {
		return nil, errors.New("analytics repository is required")
	}
	return &Service{
		repo: repo,
		now:  time.Now,
	}, nil
}

func (s *Service) GetWeeklyStats(ctx context.Context, fromSec int64, toSec int64) (analyticsdomain.WeeklyStats, error) {
	from, to, err := s.resolveRange(fromSec, toSec)
	if err != nil {
		return analyticsdomain.WeeklyStats{}, err
	}
	return s.repo.QueryWeeklyStats(normalizeContext(ctx), from, to)
}

func (s *Service) GetSummary(ctx context.Context, fromSec int64, toSec int64) (analyticsdomain.Summary, error) {
	from, to, err := s.resolveRange(fromSec, toSec)
	if err != nil {
		return analyticsdomain.Summary{}, err
	}
	return s.repo.QuerySummary(normalizeContext(ctx), from, to)
}

func (s *Service) GetTrendByDay(ctx context.Context, fromSec int64, toSec int64) (analyticsdomain.Trend, error) {
	from, to, err := s.resolveRange(fromSec, toSec)
	if err != nil {
		return analyticsdomain.Trend{}, err
	}
	return s.repo.QueryTrendByDay(normalizeContext(ctx), from, to)
}

func (s *Service) GetBreakTypeDistribution(ctx context.Context, fromSec int64, toSec int64) (analyticsdomain.BreakTypeDistribution, error) {
	from, to, err := s.resolveRange(fromSec, toSec)
	if err != nil {
		return analyticsdomain.BreakTypeDistribution{}, err
	}
	return s.repo.QueryBreakTypeDistribution(normalizeContext(ctx), from, to)
}

func (s *Service) resolveRange(fromSec int64, toSec int64) (time.Time, time.Time, error) {
	if fromSec == 0 && toSec == 0 {
		start, end := currentWeekRange(s.now())
		return start, end, nil
	}
	if toSec <= fromSec {
		return time.Time{}, time.Time{}, errors.New("invalid time range")
	}
	return time.Unix(fromSec, 0), time.Unix(toSec, 0), nil
}

func currentWeekRange(now time.Time) (time.Time, time.Time) {
	local := now.Local()
	weekday := int(local.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location()).
		AddDate(0, 0, -(weekday - 1))
	end := start.AddDate(0, 0, 7)
	return start, end
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}
