package historyadapter

import (
	"context"
	"time"

	analyticsdomain "pause/internal/backend/domain/analytics"
	"pause/internal/backend/ports"
	historydb "pause/internal/backend/storage/historydb"
)

type AnalyticsRepository struct {
	store *historydb.Store
}

var _ ports.AnalyticsRepository = (*AnalyticsRepository)(nil)

func NewAnalyticsRepository(store *historydb.Store) *AnalyticsRepository {
	return &AnalyticsRepository{store: store}
}

func (r *AnalyticsRepository) QueryWeeklyStats(ctx context.Context, from time.Time, to time.Time) (analyticsdomain.WeeklyStats, error) {
	if err := r.ensureStore(); err != nil {
		return analyticsdomain.WeeklyStats{}, err
	}
	return r.store.QueryAnalyticsWeeklyStats(ctx, from, to)
}

func (r *AnalyticsRepository) QuerySummary(ctx context.Context, from time.Time, to time.Time) (analyticsdomain.Summary, error) {
	if err := r.ensureStore(); err != nil {
		return analyticsdomain.Summary{}, err
	}
	return r.store.QueryAnalyticsSummary(ctx, from, to)
}

func (r *AnalyticsRepository) QueryTrendByDay(ctx context.Context, from time.Time, to time.Time) (analyticsdomain.Trend, error) {
	if err := r.ensureStore(); err != nil {
		return analyticsdomain.Trend{}, err
	}
	return r.store.QueryAnalyticsTrendByDay(ctx, from, to)
}

func (r *AnalyticsRepository) QueryBreakTypeDistribution(ctx context.Context, from time.Time, to time.Time) (analyticsdomain.BreakTypeDistribution, error) {
	if err := r.ensureStore(); err != nil {
		return analyticsdomain.BreakTypeDistribution{}, err
	}
	return r.store.QueryAnalyticsBreakTypeDistribution(ctx, from, to)
}

func (r *AnalyticsRepository) ensureStore() error {
	if r == nil || r.store == nil {
		return errHistoryStoreUnavailable
	}
	return nil
}
