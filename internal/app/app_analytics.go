package app

import (
	"errors"

	analyticsdomain "pause/internal/backend/domain/analytics"
)

func (a *App) GetAnalyticsWeeklyStats(fromSec int64, toSec int64) (analyticsdomain.WeeklyStats, error) {
	if a == nil || a.analytics == nil {
		return analyticsdomain.WeeklyStats{}, errors.New("analytics service unavailable")
	}
	return a.analytics.GetWeeklyStats(appContextOrBackground(a.ctx), fromSec, toSec)
}

func (a *App) GetAnalyticsSummary(fromSec int64, toSec int64) (analyticsdomain.Summary, error) {
	if a == nil || a.analytics == nil {
		return analyticsdomain.Summary{}, errors.New("analytics service unavailable")
	}
	return a.analytics.GetSummary(appContextOrBackground(a.ctx), fromSec, toSec)
}

func (a *App) GetAnalyticsTrendByDay(fromSec int64, toSec int64) (analyticsdomain.Trend, error) {
	if a == nil || a.analytics == nil {
		return analyticsdomain.Trend{}, errors.New("analytics service unavailable")
	}
	return a.analytics.GetTrendByDay(appContextOrBackground(a.ctx), fromSec, toSec)
}

func (a *App) GetAnalyticsBreakTypeDistribution(fromSec int64, toSec int64) (analyticsdomain.BreakTypeDistribution, error) {
	if a == nil || a.analytics == nil {
		return analyticsdomain.BreakTypeDistribution{}, errors.New("analytics service unavailable")
	}
	return a.analytics.GetBreakTypeDistribution(appContextOrBackground(a.ctx), fromSec, toSec)
}
