package app

import (
	"errors"
)

func (a *App) GetAnalyticsWeeklyStats(fromSec int64, toSec int64) (AnalyticsWeeklyStats, error) {
	if a == nil || a.analytics == nil {
		return AnalyticsWeeklyStats{}, errors.New("analytics service unavailable")
	}
	stats, err := a.analytics.GetWeeklyStats(appContextOrBackground(a.ctx), fromSec, toSec)
	if err != nil {
		return AnalyticsWeeklyStats{}, err
	}
	return analyticsWeeklyStatsFromDomain(stats), nil
}

func (a *App) GetAnalyticsSummary(fromSec int64, toSec int64) (AnalyticsSummary, error) {
	if a == nil || a.analytics == nil {
		return AnalyticsSummary{}, errors.New("analytics service unavailable")
	}
	summary, err := a.analytics.GetSummary(appContextOrBackground(a.ctx), fromSec, toSec)
	if err != nil {
		return AnalyticsSummary{}, err
	}
	return analyticsSummaryFromDomain(summary), nil
}

func (a *App) GetAnalyticsTrendByDay(fromSec int64, toSec int64) (AnalyticsTrend, error) {
	if a == nil || a.analytics == nil {
		return AnalyticsTrend{}, errors.New("analytics service unavailable")
	}
	trend, err := a.analytics.GetTrendByDay(appContextOrBackground(a.ctx), fromSec, toSec)
	if err != nil {
		return AnalyticsTrend{}, err
	}
	return analyticsTrendFromDomain(trend), nil
}

func (a *App) GetAnalyticsBreakTypeDistribution(fromSec int64, toSec int64) (AnalyticsBreakTypeDistribution, error) {
	if a == nil || a.analytics == nil {
		return AnalyticsBreakTypeDistribution{}, errors.New("analytics service unavailable")
	}
	distribution, err := a.analytics.GetBreakTypeDistribution(appContextOrBackground(a.ctx), fromSec, toSec)
	if err != nil {
		return AnalyticsBreakTypeDistribution{}, err
	}
	return analyticsBreakTypeDistributionFromDomain(distribution), nil
}
