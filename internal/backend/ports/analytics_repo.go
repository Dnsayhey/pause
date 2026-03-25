package ports

import (
	"context"
	"time"

	analyticsdomain "pause/internal/backend/domain/analytics"
)

type AnalyticsRepository interface {
	QueryWeeklyStats(ctx context.Context, from time.Time, to time.Time) (analyticsdomain.WeeklyStats, error)
	QuerySummary(ctx context.Context, from time.Time, to time.Time) (analyticsdomain.Summary, error)
	QueryTrendByDay(ctx context.Context, from time.Time, to time.Time) (analyticsdomain.Trend, error)
	QueryBreakTypeDistribution(ctx context.Context, from time.Time, to time.Time) (analyticsdomain.BreakTypeDistribution, error)
}
