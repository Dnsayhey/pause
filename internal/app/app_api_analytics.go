package app

import (
	"errors"

	"pause/internal/logx"
)

func (a *App) GetAnalyticsWeeklyStats(fromSec int64, toSec int64) (AnalyticsWeeklyStats, error) {
	if a == nil || a.analytics == nil {
		return AnalyticsWeeklyStats{}, errors.New("analytics service unavailable")
	}
	stats, err := a.analytics.GetWeeklyStats(appContextOrBackground(a.ctx), fromSec, toSec)
	if err != nil {
		logx.Warnf("app.analytics_weekly_stats_err from_sec=%d to_sec=%d err=%v", fromSec, toSec, err)
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
		logx.Warnf("app.analytics_summary_err from_sec=%d to_sec=%d err=%v", fromSec, toSec, err)
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
		logx.Warnf("app.analytics_trend_by_day_err from_sec=%d to_sec=%d err=%v", fromSec, toSec, err)
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
		logx.Warnf("app.analytics_break_type_distribution_err from_sec=%d to_sec=%d err=%v", fromSec, toSec, err)
		return AnalyticsBreakTypeDistribution{}, err
	}
	return analyticsBreakTypeDistributionFromDomain(distribution), nil
}
