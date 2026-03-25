package app

import (
	"errors"

	analyticsdomain "pause/internal/backend/domain/analytics"
	coreanalytics "pause/internal/core/analytics"
)

func analyticsWeeklyStatsToCore(source analyticsdomain.WeeklyStats) coreanalytics.WeeklyStats {
	reminders := make([]coreanalytics.ReminderStat, 0, len(source.Reminders))
	for _, row := range source.Reminders {
		reminders = append(reminders, coreanalytics.ReminderStat{
			ReminderID:          row.ReminderID,
			ReminderName:        row.ReminderName,
			Enabled:             row.Enabled,
			ReminderType:        row.ReminderType,
			TriggeredCount:      row.TriggeredCount,
			CompletedCount:      row.CompletedCount,
			SkippedCount:        row.SkippedCount,
			CanceledCount:       row.CanceledCount,
			TotalActualBreakSec: row.TotalActualBreakSec,
			AvgActualBreakSec:   row.AvgActualBreakSec,
		})
	}

	return coreanalytics.WeeklyStats{
		FromSec:   source.FromSec,
		ToSec:     source.ToSec,
		Reminders: reminders,
		Summary: coreanalytics.SummaryStats{
			TotalSessions:       source.Summary.TotalSessions,
			TotalCompleted:      source.Summary.TotalCompleted,
			TotalSkipped:        source.Summary.TotalSkipped,
			TotalCanceled:       source.Summary.TotalCanceled,
			TotalActualBreakSec: source.Summary.TotalActualBreakSec,
			AvgActualBreakSec:   source.Summary.AvgActualBreakSec,
		},
	}
}

func analyticsSummaryToCore(source analyticsdomain.Summary) coreanalytics.Summary {
	return coreanalytics.Summary{
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

func analyticsTrendToCore(source analyticsdomain.Trend) coreanalytics.Trend {
	points := make([]coreanalytics.TrendPoint, 0, len(source.Points))
	for _, row := range source.Points {
		points = append(points, coreanalytics.TrendPoint{
			Day:                 row.Day,
			TotalSessions:       row.TotalSessions,
			TotalCompleted:      row.TotalCompleted,
			TotalSkipped:        row.TotalSkipped,
			TotalCanceled:       row.TotalCanceled,
			CompletionRate:      row.CompletionRate,
			SkipRate:            row.SkipRate,
			TotalActualBreakSec: row.TotalActualBreakSec,
			AvgActualBreakSec:   row.AvgActualBreakSec,
		})
	}
	return coreanalytics.Trend{
		FromSec: source.FromSec,
		ToSec:   source.ToSec,
		Points:  points,
	}
}

func analyticsBreakTypeDistributionToCore(source analyticsdomain.BreakTypeDistribution) coreanalytics.BreakTypeDistribution {
	items := make([]coreanalytics.BreakTypeDistributionItem, 0, len(source.Items))
	for _, row := range source.Items {
		items = append(items, coreanalytics.BreakTypeDistributionItem{
			ReminderID:      row.ReminderID,
			ReminderName:    row.ReminderName,
			TriggeredCount:  row.TriggeredCount,
			CompletedCount:  row.CompletedCount,
			SkippedCount:    row.SkippedCount,
			CanceledCount:   row.CanceledCount,
			CompletionRate:  row.CompletionRate,
			SkipRate:        row.SkipRate,
			TriggeredShare:  row.TriggeredShare,
			ReminderType:    row.ReminderType,
			ReminderEnabled: row.ReminderEnabled,
		})
	}
	return coreanalytics.BreakTypeDistribution{
		FromSec:        source.FromSec,
		ToSec:          source.ToSec,
		TotalTriggered: source.TotalTriggered,
		Items:          items,
	}
}

func (a *App) GetAnalyticsWeeklyStats(fromSec int64, toSec int64) (coreanalytics.WeeklyStats, error) {
	if a == nil || a.analytics == nil {
		return coreanalytics.WeeklyStats{}, errors.New("analytics service unavailable")
	}
	result, err := a.analytics.GetWeeklyStats(appContextOrBackground(a.ctx), fromSec, toSec)
	if err != nil {
		return coreanalytics.WeeklyStats{}, err
	}
	return analyticsWeeklyStatsToCore(result), nil
}

func (a *App) GetAnalyticsSummary(fromSec int64, toSec int64) (coreanalytics.Summary, error) {
	if a == nil || a.analytics == nil {
		return coreanalytics.Summary{}, errors.New("analytics service unavailable")
	}
	result, err := a.analytics.GetSummary(appContextOrBackground(a.ctx), fromSec, toSec)
	if err != nil {
		return coreanalytics.Summary{}, err
	}
	return analyticsSummaryToCore(result), nil
}

func (a *App) GetAnalyticsTrendByDay(fromSec int64, toSec int64) (coreanalytics.Trend, error) {
	if a == nil || a.analytics == nil {
		return coreanalytics.Trend{}, errors.New("analytics service unavailable")
	}
	result, err := a.analytics.GetTrendByDay(appContextOrBackground(a.ctx), fromSec, toSec)
	if err != nil {
		return coreanalytics.Trend{}, err
	}
	return analyticsTrendToCore(result), nil
}

func (a *App) GetAnalyticsBreakTypeDistribution(fromSec int64, toSec int64) (coreanalytics.BreakTypeDistribution, error) {
	if a == nil || a.analytics == nil {
		return coreanalytics.BreakTypeDistribution{}, errors.New("analytics service unavailable")
	}
	result, err := a.analytics.GetBreakTypeDistribution(appContextOrBackground(a.ctx), fromSec, toSec)
	if err != nil {
		return coreanalytics.BreakTypeDistribution{}, err
	}
	return analyticsBreakTypeDistributionToCore(result), nil
}
