import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { EChartsOption } from 'echarts';
import * as echarts from 'echarts/core';
import { BarChart, HeatmapChart, LineChart } from 'echarts/charts';
import { GridComponent, LegendComponent, TooltipComponent, VisualMapComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import ReactEChartsCore from 'echarts-for-react/lib/core';
import {
  getAnalyticsBreakTypeDistribution,
  getAnalyticsHourlyHeatmap,
  getAnalyticsSummary,
  getAnalyticsTrendByDay,
  getAnalyticsWeeklyStats
} from '../api';
import { t, type Locale } from '../i18n';
import type {
  AnalyticsBreakTypeDistribution,
  AnalyticsHeatmapMetric,
  AnalyticsSummary,
  AnalyticsTrend,
  AnalyticsWeeklyStats
} from '../types';
import { GlassCard, InlineError, PillSelect } from './ui';

echarts.use([GridComponent, LegendComponent, TooltipComponent, VisualMapComponent, LineChart, BarChart, HeatmapChart, CanvasRenderer]);

type RangePreset = 'day' | 'week' | 'month';

type AnalyticsBundle = {
  summary: AnalyticsSummary;
  previousSummary: AnalyticsSummary | null;
  trend: AnalyticsTrend;
  distribution: AnalyticsBreakTypeDistribution;
  weekly: AnalyticsWeeklyStats;
  heatmap: Awaited<ReturnType<typeof getAnalyticsHourlyHeatmap>>;
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

function hourLabel(hour: number): string {
  return `${String(hour).padStart(2, '0')}:00`;
}

function metricLabel(locale: Locale, metric: AnalyticsHeatmapMetric): string {
  if (metric === 'completion_rate') return t(locale, 'analyticsMetricCompletionRate');
  if (metric === 'triggered_count') return t(locale, 'analyticsMetricTriggeredCount');
  return t(locale, 'analyticsMetricSkipRate');
}

function formatMetricValue(value: number, metric: AnalyticsHeatmapMetric): string {
  return metric === 'triggered_count' ? `${Math.round(value)}` : formatPercent(value);
}

function parseRangePreset(value: string): RangePreset {
  if (value === 'day' || value === 'week' || value === 'month') return value;
  return 'week';
}

function deltaClass(delta: number): string {
  if (delta > 0) return 'text-[#0f826b]';
  if (delta < 0) return 'text-[#b44937]';
  return 'text-[var(--text-tertiary)]';
}

export function AnalyticsPanel({ locale }: AnalyticsPanelProps) {
  const [preset, setPreset] = useState<RangePreset>('week');
  const [metric, setMetric] = useState<AnalyticsHeatmapMetric>('skip_rate');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [bundle, setBundle] = useState<AnalyticsBundle | null>(null);
  const requestIDRef = useRef(0);
  const isDarkTheme = typeof document !== 'undefined' && document.body.dataset.theme === 'dark';
  const chartTextColor = isDarkTheme ? 'rgba(232,240,248,0.84)' : 'rgba(18,34,54,0.78)';
  const chartSubtleTextColor = isDarkTheme ? 'rgba(232,240,248,0.7)' : 'rgba(18,34,54,0.65)';
  const chartGridColor = isDarkTheme ? 'rgba(198,220,236,0.18)' : 'rgba(18,34,54,0.1)';
  const chartTooltipBg = isDarkTheme ? 'rgba(14,25,36,0.94)' : 'rgba(255,255,255,0.94)';
  const chartTooltipBorder = isDarkTheme ? 'rgba(186,214,236,0.25)' : 'rgba(18,34,54,0.16)';

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
      const [summary, previousSummary, trend, distribution, weekly, heatmap] = await Promise.all([
        getAnalyticsSummary(fromSec, toSec),
        getAnalyticsSummary(previousFromSec, previousToSec).catch(() => null),
        getAnalyticsTrendByDay(fromSec, toSec),
        getAnalyticsBreakTypeDistribution(fromSec, toSec),
        getAnalyticsWeeklyStats(fromSec, toSec),
        getAnalyticsHourlyHeatmap(fromSec, toSec, metric)
      ]);
      if (requestIDRef.current !== requestID) {
        return;
      }
      setBundle({ summary, previousSummary, trend, distribution, weekly, heatmap });
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
  }, [metric, preset]);

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
      color: ['#0f826b', '#ea6b4a', '#2f73d8'],
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
  }, [bundle, chartGridColor, chartSubtleTextColor, chartTextColor, chartTooltipBg, chartTooltipBorder, locale]);

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
      color: ['#0f826b', '#ea6b4a', '#8b95a7'],
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
  }, [bundle, chartGridColor, chartSubtleTextColor, chartTextColor, chartTooltipBg, chartTooltipBorder, locale]);

  const heatmapOption = useMemo<EChartsOption | null>(() => {
    if (!bundle || bundle.heatmap.cells.length === 0) return null;

    const days = Array.from(new Set(bundle.heatmap.cells.map((cell) => cell.day))).sort();
    const hours = Array.from({ length: 24 }, (_, hour) => hour);
    const dayIndexByValue = new Map<string, number>();
    for (let i = 0; i < days.length; i += 1) {
      dayIndexByValue.set(days[i], i);
    }

    let max = 0;
    const data: Array<[number, number, number, number, number, number, number]> = [];

    for (const cell of bundle.heatmap.cells) {
      const dayIndex = dayIndexByValue.get(cell.day);
      if (dayIndex === undefined) continue;
      if (cell.value > max) max = cell.value;
      data.push([
        dayIndex,
        cell.hour,
        cell.value,
        cell.triggeredCount,
        cell.completedCount,
        cell.skippedCount,
        cell.canceledCount
      ]);
    }

    if (max <= 0) max = 1;

    return {
      animationDuration: 500,
      animationEasing: 'cubicOut',
      tooltip: {
        position: 'top',
        backgroundColor: chartTooltipBg,
        borderColor: chartTooltipBorder,
        borderWidth: 1,
        textStyle: { color: chartTextColor },
        formatter: (params) => {
          const row = (params as { data?: [number, number, number, number, number, number, number] }).data;
          if (!row) return '';
          const [dayIndex, hour, value, triggered, completed, skipped, canceled] = row;
          return [
            `${days[dayIndex]} ${hourLabel(hour)}`,
            `${metricLabel(locale, metric)}: ${formatMetricValue(value, metric)}`,
            `${t(locale, 'analyticsColTriggered')}: ${triggered}`,
            `${t(locale, 'analyticsColCompleted')}: ${completed}`,
            `${t(locale, 'analyticsColSkipped')}: ${skipped}`,
            `${t(locale, 'analyticsColCanceled')}: ${canceled}`
          ].join('<br/>');
        }
      },
      grid: {
        left: 42,
        right: 18,
        top: 16,
        bottom: 44,
        containLabel: true
      },
      xAxis: {
        type: 'category',
        data: days,
        axisLabel: { color: chartSubtleTextColor },
        axisLine: { lineStyle: { color: chartGridColor } },
        splitArea: {
          show: true
        }
      },
      yAxis: {
        type: 'category',
        data: hours.map(hourLabel),
        axisLabel: { color: chartSubtleTextColor },
        axisLine: { lineStyle: { color: chartGridColor } },
        splitArea: {
          show: true
        }
      },
      visualMap: {
        min: 0,
        max,
        calculable: true,
        orient: 'horizontal',
        left: 'center',
        bottom: 0,
        inRange: {
          color: ['#edf7f4', '#9ad9cb', '#0f826b']
        },
        textStyle: { color: chartSubtleTextColor },
        formatter: (value) => formatMetricValue(value, metric)
      },
      series: [
        {
          name: metricLabel(locale, metric),
          type: 'heatmap',
          data,
          label: {
            show: false
          },
          emphasis: {
            itemStyle: {
              shadowBlur: 10,
              shadowColor: 'rgba(0, 0, 0, 0.25)'
            }
          }
        }
      ]
    };
  }, [bundle, chartGridColor, chartSubtleTextColor, chartTextColor, chartTooltipBg, chartTooltipBorder, locale, metric]);

  const compare = bundle?.previousSummary
    ? {
        sessions: bundle.summary.totalSessions - bundle.previousSummary.totalSessions,
        completionRate: (bundle.summary.completionRate - bundle.previousSummary.completionRate) * 100,
        skipRate: (bundle.summary.skipRate - bundle.previousSummary.skipRate) * 100,
        breakTime: bundle.summary.totalActualBreakSec - bundle.previousSummary.totalActualBreakSec
      }
    : null;

  return (
    <GlassCard>
      <div className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h3 className="m-0 text-[18px]">{t(locale, 'analyticsTitle')}</h3>
          <div className="flex flex-wrap items-center gap-2">
            <PillSelect
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
            <PillSelect
              value={metric}
              onChange={(e) => {
                setMetric(e.target.value as AnalyticsHeatmapMetric);
              }}
              options={[
                { value: 'skip_rate', label: t(locale, 'analyticsMetricSkipRate') },
                { value: 'completion_rate', label: t(locale, 'analyticsMetricCompletionRate') },
                { value: 'triggered_count', label: t(locale, 'analyticsMetricTriggeredCount') }
              ]}
            />
            <button
              type="button"
              className="inline-flex appearance-none items-center justify-center border-0 bg-transparent p-0 text-[var(--control-text)] shadow-none transition-opacity hover:opacity-80 focus-visible:rounded-sm focus-visible:outline-2 focus-visible:outline-[rgba(21,123,209,0.32)] disabled:cursor-not-allowed disabled:opacity-45"
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
            <div className="grid grid-cols-2 gap-2.5 min-[760px]:grid-cols-4">
              <div className="reveal-up rounded-xl border border-[var(--surface-border)] bg-[rgba(255,255,255,0.55)] p-3">
                <p className="m-0 text-xs text-[var(--text-secondary)]">{t(locale, 'analyticsTotalSessions')}</p>
                <p className="mb-0 mt-1 text-xl font-semibold">{bundle.summary.totalSessions}</p>
                <p className={`m-0 mt-1 text-xs ${compare ? deltaClass(compare.sessions) : 'text-[var(--text-tertiary)]'}`}>
                  {compare
                    ? `${t(locale, 'analyticsComparedToPrev')}: ${formatSignedInteger(compare.sessions)}`
                    : t(locale, 'analyticsNoPrevData')}
                </p>
              </div>
              <div className="reveal-up reveal-delay-1 rounded-xl border border-[var(--surface-border)] bg-[rgba(255,255,255,0.55)] p-3">
                <p className="m-0 text-xs text-[var(--text-secondary)]">{t(locale, 'analyticsCompletionRate')}</p>
                <p className="mb-0 mt-1 text-xl font-semibold">{formatPercent(bundle.summary.completionRate)}</p>
                <p className={`m-0 mt-1 text-xs ${compare ? deltaClass(compare.completionRate) : 'text-[var(--text-tertiary)]'}`}>
                  {compare
                    ? `${t(locale, 'analyticsComparedToPrev')}: ${formatSignedPercentPoint(compare.completionRate)}`
                    : t(locale, 'analyticsNoPrevData')}
                </p>
              </div>
              <div className="reveal-up reveal-delay-2 rounded-xl border border-[var(--surface-border)] bg-[rgba(255,255,255,0.55)] p-3">
                <p className="m-0 text-xs text-[var(--text-secondary)]">{t(locale, 'analyticsSkipRate')}</p>
                <p className="mb-0 mt-1 text-xl font-semibold">{formatPercent(bundle.summary.skipRate)}</p>
                <p className={`m-0 mt-1 text-xs ${compare ? deltaClass(-compare.skipRate) : 'text-[var(--text-tertiary)]'}`}>
                  {compare
                    ? `${t(locale, 'analyticsComparedToPrev')}: ${formatSignedPercentPoint(compare.skipRate)}`
                    : t(locale, 'analyticsNoPrevData')}
                </p>
              </div>
              <div className="reveal-up reveal-delay-3 rounded-xl border border-[var(--surface-border)] bg-[rgba(255,255,255,0.55)] p-3">
                <p className="m-0 text-xs text-[var(--text-secondary)]">{t(locale, 'analyticsTotalBreakTime')}</p>
                <p className="mb-0 mt-1 text-xl font-semibold">{formatDuration(bundle.summary.totalActualBreakSec, locale)}</p>
                <p className={`m-0 mt-1 text-xs ${compare ? deltaClass(compare.breakTime) : 'text-[var(--text-tertiary)]'}`}>
                  {compare
                    ? `${t(locale, 'analyticsComparedToPrev')}: ${formatSignedDuration(compare.breakTime, locale)}`
                    : t(locale, 'analyticsNoPrevData')}
                </p>
              </div>
            </div>

            <div className="reveal-up reveal-delay-1 rounded-xl border border-[var(--surface-border)] bg-[rgba(255,255,255,0.55)] p-3">
              <p className="mb-2 mt-0 text-sm font-semibold">{t(locale, 'analyticsTrendByDay')}</p>
              {trendOption ? (
                <ReactEChartsCore echarts={echarts} option={trendOption} style={{ height: 300 }} notMerge lazyUpdate />
              ) : (
                <p className="m-0 text-sm text-[var(--text-tertiary)]">{t(locale, 'analyticsEmpty')}</p>
              )}
            </div>

            <div className="grid grid-cols-1 gap-3 min-[920px]:grid-cols-2">
              <div className="reveal-up reveal-delay-2 rounded-xl border border-[var(--surface-border)] bg-[rgba(255,255,255,0.55)] p-3">
                <p className="mb-2 mt-0 text-sm font-semibold">{t(locale, 'analyticsReminderBreakdown')}</p>
                {distributionOption ? (
                  <ReactEChartsCore echarts={echarts} option={distributionOption} style={{ height: 320 }} notMerge lazyUpdate />
                ) : (
                  <p className="m-0 text-sm text-[var(--text-tertiary)]">{t(locale, 'analyticsEmpty')}</p>
                )}
              </div>
              <div className="reveal-up reveal-delay-3 rounded-xl border border-[var(--surface-border)] bg-[rgba(255,255,255,0.55)] p-3">
                <p className="mb-2 mt-0 text-sm font-semibold">{t(locale, 'analyticsBestHours')}</p>
                {heatmapOption ? (
                  <ReactEChartsCore echarts={echarts} option={heatmapOption} style={{ height: 320 }} notMerge lazyUpdate />
                ) : (
                  <p className="m-0 text-sm text-[var(--text-tertiary)]">{t(locale, 'analyticsEmpty')}</p>
                )}
              </div>
            </div>

            <p className="m-0 text-xs text-[var(--text-tertiary)]">
              {t(locale, 'analyticsRangeLabel')}: {new Date(bundle.summary.fromSec * 1000).toLocaleString()} -{' '}
              {new Date(bundle.summary.toSec * 1000).toLocaleString()}
              {' • '}
              {t(locale, 'analyticsRowsCount')}: {bundle.weekly.reminders.length}
            </p>
          </>
        )}
      </div>
    </GlassCard>
  );
}
