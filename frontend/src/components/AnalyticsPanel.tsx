import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { EChartsOption } from 'echarts';
import * as echarts from 'echarts/core';
import { BarChart, LineChart } from 'echarts/charts';
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import ReactEChartsCore from 'echarts-for-react/lib/core';
import {
  getAnalyticsBreakTypeDistribution,
  getAnalyticsSummary,
  getAnalyticsTrendByDay,
  getAnalyticsWeeklyStats
} from '../api';
import { t, type Locale } from '../i18n';
import type {
  AnalyticsBreakTypeDistribution,
  AnalyticsSummary,
  AnalyticsTrend,
  AnalyticsWeeklyStats
} from '../types';
import { InlineError, PillSelect } from './ui';

echarts.use([GridComponent, LegendComponent, TooltipComponent, LineChart, BarChart, CanvasRenderer]);

type RangePreset = 'day' | 'week' | 'month';

type AnalyticsBundle = {
  summary: AnalyticsSummary;
  previousSummary: AnalyticsSummary | null;
  trend: AnalyticsTrend;
  distribution: AnalyticsBreakTypeDistribution;
  weekly: AnalyticsWeeklyStats;
};

type AnalyticsPanelProps = {
  locale: Locale;
};

const RANGE_SECONDS: Record<RangePreset, number> = {
  day: 24 * 60 * 60,
  week: 7 * 24 * 60 * 60,
  month: 30 * 24 * 60 * 60
};

function formatPercent(value: number): string {
  return `${(value * 100).toFixed(1)}%`;
}

function formatDuration(sec: number, locale: Locale): string {
  if (sec <= 0) {
    return locale === 'zh-CN' ? '0 秒' : '0s';
  }
  const hours = Math.floor(sec / 3600);
  const minutes = Math.floor((sec % 3600) / 60);
  const seconds = sec % 60;
  if (hours > 0) {
    if (locale === 'zh-CN') {
      return `${hours} 小时 ${minutes} 分`;
    }
    return `${hours}h ${minutes}m`;
  }
  if (minutes > 0) {
    if (locale === 'zh-CN') {
      return `${minutes} 分 ${seconds} 秒`;
    }
    return `${minutes}m ${seconds}s`;
  }
  if (locale === 'zh-CN') {
    return `${seconds} 秒`;
  }
  return `${seconds}s`;
}

function formatSignedInteger(value: number): string {
  if (value === 0) return '0';
  return `${value > 0 ? '+' : ''}${Math.round(value)}`;
}

function formatSignedPercentPoint(value: number): string {
  if (value === 0) return '0.0pp';
  return `${value > 0 ? '+' : ''}${value.toFixed(1)}pp`;
}

function formatSignedDuration(sec: number, locale: Locale): string {
  if (sec === 0) {
    return locale === 'zh-CN' ? '0 秒' : '0s';
  }
  return `${sec > 0 ? '+' : '-'}${formatDuration(Math.abs(sec), locale)}`;
}

function computeRange(preset: RangePreset): { fromSec: number; toSec: number } {
  const toSec = Math.floor(Date.now() / 1000);
  return {
    fromSec: toSec - RANGE_SECONDS[preset],
    toSec
  };
}

function parseRangePreset(value: string): RangePreset {
  if (value === 'day' || value === 'week' || value === 'month') return value;
  return 'week';
}

function resolveThemeColor(variableName: string, fallback: string): string {
  if (typeof window === 'undefined' || typeof document === 'undefined') {
    return fallback;
  }
  const target = document.body ?? document.documentElement;
  const value = window.getComputedStyle(target).getPropertyValue(variableName).trim();
  return value || fallback;
}

function deltaClass(delta: number): string {
  if (delta > 0) return 'text-[var(--positive-text)]';
  if (delta < 0) return 'text-[var(--negative-text)]';
  return 'text-[var(--text-tertiary)]';
}

export function AnalyticsPanel({ locale }: AnalyticsPanelProps) {
  const [preset, setPreset] = useState<RangePreset>('week');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [bundle, setBundle] = useState<AnalyticsBundle | null>(null);
  const requestIDRef = useRef(0);
  const chartTextColor = resolveThemeColor('--chart-text-color', 'rgba(22, 33, 29, 0.82)');
  const chartSubtleTextColor = resolveThemeColor('--chart-subtle-text-color', 'rgba(22, 33, 29, 0.62)');
  const chartGridColor = resolveThemeColor('--chart-grid-color', 'rgba(22, 33, 29, 0.13)');
  const chartTooltipBg = resolveThemeColor('--chart-tooltip-bg', 'rgba(255, 255, 255, 0.96)');
  const chartTooltipBorder = resolveThemeColor('--chart-tooltip-border', 'rgba(22, 33, 29, 0.16)');
  const chartSeries1 = resolveThemeColor('--chart-series-1', '#2f7a66');
  const chartSeries2 = resolveThemeColor('--chart-series-2', '#d07b58');
  const chartSeries3 = resolveThemeColor('--chart-series-3', '#5f78ad');

  const loadAnalytics = useCallback(async () => {
    const { fromSec, toSec } = computeRange(preset);
    const requestID = requestIDRef.current + 1;
    requestIDRef.current = requestID;
    setLoading(true);
    setError('');

    const periodSec = toSec - fromSec;
    const previousFromSec = fromSec - periodSec;
    const previousToSec = fromSec;

    try {
      const [summary, previousSummary, trend, distribution, weekly] = await Promise.all([
        getAnalyticsSummary(fromSec, toSec),
        getAnalyticsSummary(previousFromSec, previousToSec).catch(() => null),
        getAnalyticsTrendByDay(fromSec, toSec),
        getAnalyticsBreakTypeDistribution(fromSec, toSec),
        getAnalyticsWeeklyStats(fromSec, toSec)
      ]);
      if (requestIDRef.current !== requestID) {
        return;
      }
      setBundle({ summary, previousSummary, trend, distribution, weekly });
    } catch (err) {
      if (requestIDRef.current !== requestID) {
        return;
      }
      setError(String(err));
    } finally {
      if (requestIDRef.current === requestID) {
        setLoading(false);
      }
    }
  }, [preset]);

  useEffect(() => {
    void loadAnalytics();
  }, [loadAnalytics]);

  const trendOption = useMemo<EChartsOption | null>(() => {
    if (!bundle || bundle.trend.points.length === 0) return null;
    const days = bundle.trend.points.map((point) => point.day);
    const sessions = bundle.trend.points.map((point) => point.totalSessions);
    const completionRates = bundle.trend.points.map((point) => Number((point.completionRate * 100).toFixed(2)));
    const skipRates = bundle.trend.points.map((point) => Number((point.skipRate * 100).toFixed(2)));

    return {
      animationDuration: 520,
      animationEasing: 'cubicOut',
      color: [chartSeries1, chartSeries2, chartSeries3],
      tooltip: {
        trigger: 'axis',
        backgroundColor: chartTooltipBg,
        borderColor: chartTooltipBorder,
        borderWidth: 1,
        textStyle: { color: chartTextColor }
      },
      legend: {
        data: [t(locale, 'analyticsTotalSessions'), t(locale, 'analyticsSkipRate'), t(locale, 'analyticsCompletionRate')],
        textStyle: { color: chartTextColor }
      },
      grid: {
        left: 26,
        right: 30,
        top: 36,
        bottom: 18,
        containLabel: true
      },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: days,
        axisLabel: { color: chartSubtleTextColor },
        axisLine: { lineStyle: { color: chartGridColor } }
      },
      yAxis: [
        {
          type: 'value',
          name: t(locale, 'analyticsTotalSessions'),
          nameTextStyle: { color: chartSubtleTextColor },
          axisLabel: { color: chartSubtleTextColor },
          axisLine: { lineStyle: { color: chartGridColor } },
          splitLine: {
            lineStyle: { color: chartGridColor }
          }
        },
        {
          type: 'value',
          name: '%',
          min: 0,
          max: 100,
          nameTextStyle: { color: chartSubtleTextColor },
          axisLabel: { color: chartSubtleTextColor, formatter: '{value}%' },
          axisLine: { lineStyle: { color: chartGridColor } }
        }
      ],
      series: [
        {
          name: t(locale, 'analyticsTotalSessions'),
          type: 'line',
          smooth: true,
          lineStyle: { width: 3 },
          areaStyle: { opacity: 0.14 },
          emphasis: { focus: 'series' },
          data: sessions
        },
        {
          name: t(locale, 'analyticsSkipRate'),
          type: 'line',
          smooth: true,
          yAxisIndex: 1,
          data: skipRates
        },
        {
          name: t(locale, 'analyticsCompletionRate'),
          type: 'line',
          smooth: true,
          yAxisIndex: 1,
          data: completionRates
        }
      ]
    };
  }, [
    bundle,
    chartGridColor,
    chartSeries1,
    chartSeries2,
    chartSeries3,
    chartSubtleTextColor,
    chartTextColor,
    chartTooltipBg,
    chartTooltipBorder,
    locale
  ]);

  const distributionOption = useMemo<EChartsOption | null>(() => {
    if (!bundle || bundle.distribution.items.length === 0) return null;
    const items = bundle.distribution.items.slice(0, 8);
    const categories = items.map((item) => {
      if (!item.deliveryType) return item.reminderName;
      return `${item.reminderName} (${item.deliveryType})`;
    });
    return {
      animationDuration: 500,
      animationEasing: 'cubicOut',
      color: [chartSeries1, chartSeries2, chartSeries3],
      tooltip: {
        trigger: 'axis',
        axisPointer: {
          type: 'shadow'
        },
        backgroundColor: chartTooltipBg,
        borderColor: chartTooltipBorder,
        borderWidth: 1,
        textStyle: { color: chartTextColor }
      },
      legend: {
        data: [t(locale, 'analyticsColCompleted'), t(locale, 'analyticsColSkipped'), t(locale, 'analyticsColCanceled')],
        textStyle: { color: chartTextColor }
      },
      grid: {
        left: 110,
        right: 26,
        top: 36,
        bottom: 14,
        containLabel: true
      },
      xAxis: {
        type: 'value',
        axisLabel: { color: chartSubtleTextColor },
        axisLine: { lineStyle: { color: chartGridColor } },
        splitLine: {
          lineStyle: { color: chartGridColor }
        }
      },
      yAxis: {
        type: 'category',
        data: categories,
        axisLabel: { color: chartSubtleTextColor },
        axisLine: { lineStyle: { color: chartGridColor } }
      },
      series: [
        {
          name: t(locale, 'analyticsColCompleted'),
          type: 'bar',
          stack: 'total',
          data: items.map((item) => item.completedCount)
        },
        {
          name: t(locale, 'analyticsColSkipped'),
          type: 'bar',
          stack: 'total',
          data: items.map((item) => item.skippedCount)
        },
        {
          name: t(locale, 'analyticsColCanceled'),
          type: 'bar',
          stack: 'total',
          data: items.map((item) => item.canceledCount)
        }
      ]
    };
  }, [
    bundle,
    chartGridColor,
    chartSeries1,
    chartSeries2,
    chartSeries3,
    chartSubtleTextColor,
    chartTextColor,
    chartTooltipBg,
    chartTooltipBorder,
    locale
  ]);

  const compare = bundle?.previousSummary
    ? {
        sessions: bundle.summary.totalSessions - bundle.previousSummary.totalSessions,
        completionRate: (bundle.summary.completionRate - bundle.previousSummary.completionRate) * 100,
        skipRate: (bundle.summary.skipRate - bundle.previousSummary.skipRate) * 100,
        breakTime: bundle.summary.totalActualBreakSec - bundle.previousSummary.totalActualBreakSec
      }
    : null;

  return (
    <section className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h3 className="m-0 text-[18px]">{t(locale, 'analyticsTitle')}</h3>
          <div className="flex flex-wrap items-center gap-2">
            <PillSelect
              size="compact"
              className="min-w-[96px] w-[96px]"
              value={preset}
              onChange={(e) => {
                setPreset(parseRangePreset(e.target.value));
              }}
              options={[
                { value: 'day', label: t(locale, 'analyticsRangeDay') },
                { value: 'week', label: t(locale, 'analyticsRangeWeek') },
                { value: 'month', label: t(locale, 'analyticsRangeMonth') }
              ]}
            />
            <button
              type="button"
              className="inline-flex appearance-none items-center justify-center border-0 bg-transparent p-0 text-[var(--control-text)] shadow-none transition-opacity hover:opacity-80 focus-visible:rounded-sm focus-visible:outline-2 focus-visible:outline-[var(--accent-ring)] disabled:cursor-not-allowed disabled:opacity-45"
              onClick={() => void loadAnalytics()}
              disabled={loading}
              aria-label={loading ? t(locale, 'analyticsRefreshing') : t(locale, 'analyticsRefresh')}
              title={loading ? t(locale, 'analyticsRefreshing') : t(locale, 'analyticsRefresh')}
            >
              <svg
                viewBox="0 0 24 24"
                className={`h-[18px] w-[18px] ${loading ? 'animate-spin' : ''}`}
                fill="none"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  d="M20 12a8 8 0 1 1-2.34-5.66M20 4v4h-4"
                  stroke="currentColor"
                  strokeWidth="2.1"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            </button>
          </div>
        </div>

        {error && <InlineError message={error} />}

        {!bundle ? (
          <p className="m-0 text-sm text-[var(--text-secondary)]">{t(locale, 'analyticsLoading')}</p>
        ) : (
          <>
            <div className="reveal-up grid grid-cols-2 gap-x-4 gap-y-3 min-[760px]:grid-cols-4 min-[760px]:gap-x-6">
              <div className="min-w-0">
                <p className="m-0 text-xs text-[var(--text-secondary)]">{t(locale, 'analyticsTotalSessions')}</p>
                <p className="mb-0 mt-1 text-xl font-semibold">{bundle.summary.totalSessions}</p>
                <p className={`m-0 mt-1 text-xs ${compare ? deltaClass(compare.sessions) : 'text-[var(--text-tertiary)]'}`}>
                  {compare
                    ? `${t(locale, 'analyticsComparedToPrev')}: ${formatSignedInteger(compare.sessions)}`
                    : t(locale, 'analyticsNoPrevData')}
                </p>
              </div>
              <div className="min-w-0">
                <p className="m-0 text-xs text-[var(--text-secondary)]">{t(locale, 'analyticsCompletionRate')}</p>
                <p className="mb-0 mt-1 text-xl font-semibold">{formatPercent(bundle.summary.completionRate)}</p>
                <p className={`m-0 mt-1 text-xs ${compare ? deltaClass(compare.completionRate) : 'text-[var(--text-tertiary)]'}`}>
                  {compare
                    ? `${t(locale, 'analyticsComparedToPrev')}: ${formatSignedPercentPoint(compare.completionRate)}`
                    : t(locale, 'analyticsNoPrevData')}
                </p>
              </div>
              <div className="min-w-0">
                <p className="m-0 text-xs text-[var(--text-secondary)]">{t(locale, 'analyticsSkipRate')}</p>
                <p className="mb-0 mt-1 text-xl font-semibold">{formatPercent(bundle.summary.skipRate)}</p>
                <p className={`m-0 mt-1 text-xs ${compare ? deltaClass(-compare.skipRate) : 'text-[var(--text-tertiary)]'}`}>
                  {compare
                    ? `${t(locale, 'analyticsComparedToPrev')}: ${formatSignedPercentPoint(compare.skipRate)}`
                    : t(locale, 'analyticsNoPrevData')}
                </p>
              </div>
              <div className="min-w-0">
                <p className="m-0 text-xs text-[var(--text-secondary)]">{t(locale, 'analyticsTotalBreakTime')}</p>
                <p className="mb-0 mt-1 text-xl font-semibold">{formatDuration(bundle.summary.totalActualBreakSec, locale)}</p>
                <p className={`m-0 mt-1 text-xs ${compare ? deltaClass(compare.breakTime) : 'text-[var(--text-tertiary)]'}`}>
                  {compare
                    ? `${t(locale, 'analyticsComparedToPrev')}: ${formatSignedDuration(compare.breakTime, locale)}`
                    : t(locale, 'analyticsNoPrevData')}
                </p>
              </div>
            </div>

            <div className="reveal-up reveal-delay-1 pt-1">
              <p className="mb-2 mt-0 text-sm font-semibold">{t(locale, 'analyticsTrendByDay')}</p>
              {trendOption ? (
                <ReactEChartsCore echarts={echarts} option={trendOption} style={{ height: 300 }} notMerge lazyUpdate />
              ) : (
                <p className="m-0 text-sm text-[var(--text-tertiary)]">{t(locale, 'analyticsEmpty')}</p>
              )}
            </div>

            <div className="reveal-up reveal-delay-2 pt-1">
              <p className="mb-2 mt-0 text-sm font-semibold">{t(locale, 'analyticsReminderBreakdown')}</p>
              {distributionOption ? (
                <ReactEChartsCore echarts={echarts} option={distributionOption} style={{ height: 320 }} notMerge lazyUpdate />
              ) : (
                <p className="m-0 text-sm text-[var(--text-tertiary)]">{t(locale, 'analyticsEmpty')}</p>
              )}
            </div>

            <p className="m-0 text-xs text-[var(--text-tertiary)]">
              {t(locale, 'analyticsRangeLabel')}: {new Date(bundle.summary.fromSec * 1000).toLocaleString()} -{' '}
              {new Date(bundle.summary.toSec * 1000).toLocaleString()}
              {' • '}
              {t(locale, 'analyticsRowsCount')}: {bundle.weekly.reminders.length}
            </p>
          </>
        )}
    </section>
  );
}
