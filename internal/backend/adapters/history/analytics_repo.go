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
	_ = ctx
	if err := r.ensureStore(); err != nil {
		return analyticsdomain.WeeklyStats{}, err
	}
	stats, err := r.store.QueryAnalyticsWeeklyStats(from, to)
	if err != nil {
		return analyticsdomain.WeeklyStats{}, err
	}
	return weeklyStatsFromHistory(stats), nil
}

func (r *AnalyticsRepository) QuerySummary(ctx context.Context, from time.Time, to time.Time) (analyticsdomain.Summary, error) {
	_ = ctx
	if err := r.ensureStore(); err != nil {
		return analyticsdomain.Summary{}, err
	}
	summary, err := r.store.QueryAnalyticsSummary(from, to)
	if err != nil {
		return analyticsdomain.Summary{}, err
	}
	return summaryFromHistory(summary), nil
}

func (r *AnalyticsRepository) QueryTrendByDay(ctx context.Context, from time.Time, to time.Time) (analyticsdomain.Trend, error) {
	_ = ctx
	if err := r.ensureStore(); err != nil {
		return analyticsdomain.Trend{}, err
	}
	trend, err := r.store.QueryAnalyticsTrendByDay(from, to)
	if err != nil {
		return analyticsdomain.Trend{}, err
	}
	return trendFromHistory(trend), nil
}

func (r *AnalyticsRepository) QueryBreakTypeDistribution(ctx context.Context, from time.Time, to time.Time) (analyticsdomain.BreakTypeDistribution, error) {
	_ = ctx
	if err := r.ensureStore(); err != nil {
		return analyticsdomain.BreakTypeDistribution{}, err
	}
	distribution, err := r.store.QueryAnalyticsBreakTypeDistribution(from, to)
	if err != nil {
		return analyticsdomain.BreakTypeDistribution{}, err
	}
	return breakTypeDistributionFromHistory(distribution), nil
}

func (r *AnalyticsRepository) ensureStore() error {
	if r == nil || r.store == nil {
		return errHistoryStoreUnavailable
	}
	return nil
}

func weeklyStatsFromHistory(source historydb.AnalyticsWeeklyStats) analyticsdomain.WeeklyStats {
	items := make([]analyticsdomain.ReminderStat, 0, len(source.Reminders))
	for _, item := range source.Reminders {
		items = append(items, analyticsdomain.ReminderStat{
			ReminderID:          item.ReminderID,
			ReminderName:        item.ReminderName,
			Enabled:             item.Enabled,
			ReminderType:        item.ReminderType,
			TriggeredCount:      item.TriggeredCount,
			CompletedCount:      item.CompletedCount,
			SkippedCount:        item.SkippedCount,
			CanceledCount:       item.CanceledCount,
			TotalActualBreakSec: item.TotalActualBreakSec,
			AvgActualBreakSec:   item.AvgActualBreakSec,
		})
	}
	return analyticsdomain.WeeklyStats{
		FromSec:   source.FromSec,
		ToSec:     source.ToSec,
		Reminders: items,
		Summary: analyticsdomain.SummaryStats{
			TotalSessions:       source.Summary.TotalSessions,
			TotalCompleted:      source.Summary.TotalCompleted,
			TotalSkipped:        source.Summary.TotalSkipped,
			TotalCanceled:       source.Summary.TotalCanceled,
			TotalActualBreakSec: source.Summary.TotalActualBreakSec,
			AvgActualBreakSec:   source.Summary.AvgActualBreakSec,
		},
	}
}

func summaryFromHistory(source historydb.AnalyticsSummary) analyticsdomain.Summary {
	return analyticsdomain.Summary{
		FromSec:             source.FromSec,
		ToSec:               source.ToSec,
		TotalSessions:       source.TotalSessions,
		TotalCompleted:      source.TotalCompleted,
		TotalSkipped:        source.TotalSkipped,
		TotalCanceled:       source.TotalCanceled,
		CompletionRate:      source.CompletionRate,
		SkipRate:            source.SkipRate,
		TotalActualBreakSec: source.TotalActualBreakSec,
		AvgActualBreakSec:   source.AvgActualBreakSec,
	}
}

func trendFromHistory(source historydb.AnalyticsTrend) analyticsdomain.Trend {
	points := make([]analyticsdomain.TrendPoint, 0, len(source.Points))
	for _, item := range source.Points {
		points = append(points, analyticsdomain.TrendPoint{
			Day:                 item.Day,
			TotalSessions:       item.TotalSessions,
			TotalCompleted:      item.TotalCompleted,
			TotalSkipped:        item.TotalSkipped,
			TotalCanceled:       item.TotalCanceled,
			CompletionRate:      item.CompletionRate,
			SkipRate:            item.SkipRate,
			TotalActualBreakSec: item.TotalActualBreakSec,
			AvgActualBreakSec:   item.AvgActualBreakSec,
		})
	}
	return analyticsdomain.Trend{
		FromSec: source.FromSec,
		ToSec:   source.ToSec,
		Points:  points,
	}
}

func breakTypeDistributionFromHistory(source historydb.AnalyticsBreakTypeDistribution) analyticsdomain.BreakTypeDistribution {
	items := make([]analyticsdomain.BreakTypeDistributionItem, 0, len(source.Items))
	for _, item := range source.Items {
		items = append(items, analyticsdomain.BreakTypeDistributionItem{
			ReminderID:      item.ReminderID,
			ReminderName:    item.ReminderName,
			TriggeredCount:  item.TriggeredCount,
			CompletedCount:  item.CompletedCount,
			SkippedCount:    item.SkippedCount,
			CanceledCount:   item.CanceledCount,
			CompletionRate:  item.CompletionRate,
			SkipRate:        item.SkipRate,
			TriggeredShare:  item.TriggeredShare,
			ReminderType:    item.ReminderType,
			ReminderEnabled: item.ReminderEnabled,
		})
	}
	return analyticsdomain.BreakTypeDistribution{
		FromSec:        source.FromSec,
		ToSec:          source.ToSec,
		TotalTriggered: source.TotalTriggered,
		Items:          items,
	}
}
