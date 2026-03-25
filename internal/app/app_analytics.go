package app

import (
	"errors"

	analyticsdomain "pause/internal/backend/domain/analytics"
	"pause/internal/core/history"
)

func analyticsWeeklyStatsToHistory(source analyticsdomain.WeeklyStats) history.AnalyticsWeeklyStats {
	reminders := make([]history.AnalyticsReminderStat, 0, len(source.Reminders))
	for _, row := range source.Reminders {
		reminders = append(reminders, history.AnalyticsReminderStat{
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

	return history.AnalyticsWeeklyStats{
		FromSec:   source.FromSec,
		ToSec:     source.ToSec,
		Reminders: reminders,
		Summary: history.AnalyticsSummaryStats{
			TotalSessions:       source.Summary.TotalSessions,
			TotalCompleted:      source.Summary.TotalCompleted,
			TotalSkipped:        source.Summary.TotalSkipped,
			TotalCanceled:       source.Summary.TotalCanceled,
			TotalActualBreakSec: source.Summary.TotalActualBreakSec,
			AvgActualBreakSec:   source.Summary.AvgActualBreakSec,
		},
	}
}

func analyticsSummaryToHistory(source analyticsdomain.Summary) history.AnalyticsSummary {
	return history.AnalyticsSummary{
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

func analyticsTrendToHistory(source analyticsdomain.Trend) history.AnalyticsTrend {
	points := make([]history.AnalyticsTrendPoint, 0, len(source.Points))
	for _, row := range source.Points {
		points = append(points, history.AnalyticsTrendPoint{
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
	return history.AnalyticsTrend{
		FromSec: source.FromSec,
		ToSec:   source.ToSec,
		Points:  points,
	}
}

func analyticsBreakTypeDistributionToHistory(source analyticsdomain.BreakTypeDistribution) history.AnalyticsBreakTypeDistribution {
	items := make([]history.AnalyticsBreakTypeDistributionItem, 0, len(source.Items))
	for _, row := range source.Items {
		items = append(items, history.AnalyticsBreakTypeDistributionItem{
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
	return history.AnalyticsBreakTypeDistribution{
		FromSec:        source.FromSec,
		ToSec:          source.ToSec,
		TotalTriggered: source.TotalTriggered,
		Items:          items,
	}
}

func (a *App) GetAnalyticsWeeklyStats(fromSec int64, toSec int64) (history.AnalyticsWeeklyStats, error) {
	if a == nil || a.analytics == nil {
		return history.AnalyticsWeeklyStats{}, errors.New("analytics service unavailable")
	}
	result, err := a.analytics.GetWeeklyStats(appContextOrBackground(a.ctx), fromSec, toSec)
	if err != nil {
		return history.AnalyticsWeeklyStats{}, err
	}
	return analyticsWeeklyStatsToHistory(result), nil
}

func (a *App) GetAnalyticsSummary(fromSec int64, toSec int64) (history.AnalyticsSummary, error) {
	if a == nil || a.analytics == nil {
		return history.AnalyticsSummary{}, errors.New("analytics service unavailable")
	}
	result, err := a.analytics.GetSummary(appContextOrBackground(a.ctx), fromSec, toSec)
	if err != nil {
		return history.AnalyticsSummary{}, err
	}
	return analyticsSummaryToHistory(result), nil
}

func (a *App) GetAnalyticsTrendByDay(fromSec int64, toSec int64) (history.AnalyticsTrend, error) {
	if a == nil || a.analytics == nil {
		return history.AnalyticsTrend{}, errors.New("analytics service unavailable")
	}
	result, err := a.analytics.GetTrendByDay(appContextOrBackground(a.ctx), fromSec, toSec)
	if err != nil {
		return history.AnalyticsTrend{}, err
	}
	return analyticsTrendToHistory(result), nil
}

func (a *App) GetAnalyticsBreakTypeDistribution(fromSec int64, toSec int64) (history.AnalyticsBreakTypeDistribution, error) {
	if a == nil || a.analytics == nil {
		return history.AnalyticsBreakTypeDistribution{}, errors.New("analytics service unavailable")
	}
	result, err := a.analytics.GetBreakTypeDistribution(appContextOrBackground(a.ctx), fromSec, toSec)
	if err != nil {
		return history.AnalyticsBreakTypeDistribution{}, err
	}
	return analyticsBreakTypeDistributionToHistory(result), nil
}
