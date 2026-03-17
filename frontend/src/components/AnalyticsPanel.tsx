import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
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
  AnalyticsHourlyHeatmap,
  AnalyticsSummary,
  AnalyticsTrend,
  AnalyticsWeeklyStats
} from '../types';
import { GlassCard, InlineError, PillSelect, PrimaryButton } from './ui';

type RangePreset = 'day' | 'week' | 'month';

type AnalyticsBundle = {
  summary: AnalyticsSummary;
  trend: AnalyticsTrend;
  distribution: AnalyticsBreakTypeDistribution;
  weekly: AnalyticsWeeklyStats;
  heatmap: AnalyticsHourlyHeatmap;
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

type HourAggregate = {
  hour: number;
  triggeredCount: number;
  completedCount: number;
  skippedCount: number;
  canceledCount: number;
  value: number;
};

function aggregateHours(heatmap: AnalyticsHourlyHeatmap, metric: AnalyticsHeatmapMetric): HourAggregate[] {
  const byHour = new Map<number, Omit<HourAggregate, 'value'>>();
  for (const cell of heatmap.cells) {
    const prev = byHour.get(cell.hour);
    if (prev) {
      prev.triggeredCount += cell.triggeredCount;
      prev.completedCount += cell.completedCount;
      prev.skippedCount += cell.skippedCount;
      prev.canceledCount += cell.canceledCount;
      continue;
    }
    byHour.set(cell.hour, {
      hour: cell.hour,
      triggeredCount: cell.triggeredCount,
      completedCount: cell.completedCount,
      skippedCount: cell.skippedCount,
      canceledCount: cell.canceledCount
    });
  }

  const rows: HourAggregate[] = [];
  for (let hour = 0; hour < 24; hour += 1) {
    const base =
      byHour.get(hour) ??
      ({
        hour,
        triggeredCount: 0,
        completedCount: 0,
        skippedCount: 0,
        canceledCount: 0
      } as Omit<HourAggregate, 'value'>);

    let value = 0;
    if (metric === 'triggered_count') {
      value = base.triggeredCount;
    } else if (base.triggeredCount > 0) {
      value = metric === 'completion_rate' ? base.completedCount / base.triggeredCount : base.skippedCount / base.triggeredCount;
    }

    rows.push({
      ...base,
      value
    });
  }
  return rows.sort((a, b) => b.value - a.value || b.triggeredCount - a.triggeredCount || a.hour - b.hour);
}

export function AnalyticsPanel({ locale }: AnalyticsPanelProps) {
  const [preset, setPreset] = useState<RangePreset>('week');
  const [metric, setMetric] = useState<AnalyticsHeatmapMetric>('skip_rate');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [bundle, setBundle] = useState<AnalyticsBundle | null>(null);
  const requestIDRef = useRef(0);

  const loadAnalytics = useCallback(async () => {
    const { fromSec, toSec } = computeRange(preset);
    const requestID = requestIDRef.current + 1;
    requestIDRef.current = requestID;
    setLoading(true);
    setError('');

    try {
      const [summary, trend, distribution, weekly, heatmap] = await Promise.all([
        getAnalyticsSummary(fromSec, toSec),
        getAnalyticsTrendByDay(fromSec, toSec),
        getAnalyticsBreakTypeDistribution(fromSec, toSec),
        getAnalyticsWeeklyStats(fromSec, toSec),
        getAnalyticsHourlyHeatmap(fromSec, toSec, metric)
      ]);
      if (requestIDRef.current !== requestID) {
        return;
      }
      setBundle({ summary, trend, distribution, weekly, heatmap });
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

  const topHours = useMemo(() => {
    if (!bundle) return [];
    return aggregateHours(bundle.heatmap, metric).slice(0, 8);
  }, [bundle, metric]);

  const maxHourValue = useMemo(() => {
    let max = 0;
    for (const row of topHours) {
      if (row.value > max) {
        max = row.value;
      }
    }
    return max;
  }, [topHours]);

  return (
    <GlassCard>
      <div className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h3 className="m-0 text-[18px]">{t(locale, 'analyticsTitle')}</h3>
          <div className="flex flex-wrap items-center gap-2">
            <div className="inline-flex rounded-full border border-[var(--glass-control-border)] bg-[var(--glass-control-bg)] p-1 [backdrop-filter:blur(var(--surface-blur))_saturate(var(--surface-sat))]">
              {(['day', 'week', 'month'] as const).map((option) => {
                const active = preset === option;
                return (
                  <button
                    key={option}
                    type="button"
                    className={`cursor-pointer rounded-full px-3 py-1 text-xs ${
                      active ? 'bg-[#0f826b] text-white' : 'bg-transparent text-[#122236]'
                    }`}
                    onClick={() => {
                      setPreset(option);
                    }}
                  >
                    {t(locale, option === 'day' ? 'analyticsRangeDay' : option === 'week' ? 'analyticsRangeWeek' : 'analyticsRangeMonth')}
                  </button>
                );
              })}
            </div>
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
            <PrimaryButton className="min-h-[28px] px-3 py-1 text-xs" onClick={() => void loadAnalytics()} disabled={loading}>
              {loading ? t(locale, 'analyticsRefreshing') : t(locale, 'analyticsRefresh')}
            </PrimaryButton>
          </div>
        </div>

        {error && <InlineError message={error} />}

        {!bundle ? (
          <p className="m-0 text-sm text-[rgba(18,34,54,0.7)]">{t(locale, 'analyticsLoading')}</p>
        ) : (
          <>
            <div className="grid grid-cols-2 gap-2.5 min-[760px]:grid-cols-4">
              <div className="rounded-xl border border-[var(--surface-border)] bg-[rgba(255,255,255,0.55)] p-3">
                <p className="m-0 text-xs text-[rgba(18,34,54,0.7)]">{t(locale, 'analyticsTotalSessions')}</p>
                <p className="mb-0 mt-1 text-xl font-semibold">{bundle.summary.totalSessions}</p>
              </div>
              <div className="rounded-xl border border-[var(--surface-border)] bg-[rgba(255,255,255,0.55)] p-3">
                <p className="m-0 text-xs text-[rgba(18,34,54,0.7)]">{t(locale, 'analyticsCompletionRate')}</p>
                <p className="mb-0 mt-1 text-xl font-semibold">{formatPercent(bundle.summary.completionRate)}</p>
              </div>
              <div className="rounded-xl border border-[var(--surface-border)] bg-[rgba(255,255,255,0.55)] p-3">
                <p className="m-0 text-xs text-[rgba(18,34,54,0.7)]">{t(locale, 'analyticsSkipRate')}</p>
                <p className="mb-0 mt-1 text-xl font-semibold">{formatPercent(bundle.summary.skipRate)}</p>
              </div>
              <div className="rounded-xl border border-[var(--surface-border)] bg-[rgba(255,255,255,0.55)] p-3">
                <p className="m-0 text-xs text-[rgba(18,34,54,0.7)]">{t(locale, 'analyticsTotalBreakTime')}</p>
                <p className="mb-0 mt-1 text-xl font-semibold">{formatDuration(bundle.summary.totalActualBreakSec, locale)}</p>
              </div>
            </div>

            <div>
              <p className="mb-2 mt-1 text-sm font-semibold">{t(locale, 'analyticsTrendByDay')}</p>
              <div className="overflow-x-auto rounded-xl border border-[var(--surface-border)]">
                <table className="w-full border-collapse text-left text-xs">
                  <thead className="bg-[rgba(18,34,54,0.06)]">
                    <tr>
                      <th className="px-3 py-2">{t(locale, 'analyticsColDay')}</th>
                      <th className="px-3 py-2">{t(locale, 'analyticsColTriggered')}</th>
                      <th className="px-3 py-2">{t(locale, 'analyticsColCompleted')}</th>
                      <th className="px-3 py-2">{t(locale, 'analyticsColSkipped')}</th>
                      <th className="px-3 py-2">{t(locale, 'analyticsColSkipRate')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {bundle.trend.points.length === 0 ? (
                      <tr>
                        <td colSpan={5} className="px-3 py-4 text-center text-[rgba(18,34,54,0.65)]">
                          {t(locale, 'analyticsEmpty')}
                        </td>
                      </tr>
                    ) : (
                      bundle.trend.points.map((point) => (
                        <tr key={point.day} className="border-t border-[rgba(18,34,54,0.08)]">
                          <td className="px-3 py-2">{point.day}</td>
                          <td className="px-3 py-2">{point.totalSessions}</td>
                          <td className="px-3 py-2">{point.totalCompleted}</td>
                          <td className="px-3 py-2">{point.totalSkipped}</td>
                          <td className="px-3 py-2">{formatPercent(point.skipRate)}</td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>

            <div>
              <p className="mb-2 mt-1 text-sm font-semibold">{t(locale, 'analyticsReminderBreakdown')}</p>
              <div className="overflow-x-auto rounded-xl border border-[var(--surface-border)]">
                <table className="w-full border-collapse text-left text-xs">
                  <thead className="bg-[rgba(18,34,54,0.06)]">
                    <tr>
                      <th className="px-3 py-2">{t(locale, 'analyticsColReminder')}</th>
                      <th className="px-3 py-2">{t(locale, 'analyticsColTriggered')}</th>
                      <th className="px-3 py-2">{t(locale, 'analyticsColCompleted')}</th>
                      <th className="px-3 py-2">{t(locale, 'analyticsColSkipped')}</th>
                      <th className="px-3 py-2">{t(locale, 'analyticsColCanceled')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {bundle.distribution.items.length === 0 ? (
                      <tr>
                        <td colSpan={5} className="px-3 py-4 text-center text-[rgba(18,34,54,0.65)]">
                          {t(locale, 'analyticsEmpty')}
                        </td>
                      </tr>
                    ) : (
                      bundle.distribution.items.map((item) => (
                        <tr key={`${item.reminderId}-${item.reminderName}-${item.deliveryType ?? ''}`} className="border-t border-[rgba(18,34,54,0.08)]">
                          <td className="px-3 py-2">{item.reminderName}</td>
                          <td className="px-3 py-2">{item.triggeredCount}</td>
                          <td className="px-3 py-2">{item.completedCount}</td>
                          <td className="px-3 py-2">{item.skippedCount}</td>
                          <td className="px-3 py-2">{item.canceledCount}</td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>

            <div>
              <p className="mb-2 mt-1 text-sm font-semibold">{t(locale, 'analyticsBestHours')}</p>
              <div className="grid gap-2">
                {topHours.map((row) => {
                  const width = maxHourValue > 0 ? (row.value / maxHourValue) * 100 : 0;
                  return (
                    <div key={row.hour} className="rounded-lg border border-[var(--surface-border)] bg-[rgba(255,255,255,0.5)] p-2">
                      <div className="mb-1 flex items-center justify-between text-xs">
                        <span>{hourLabel(row.hour)}</span>
                        <span className="text-[rgba(18,34,54,0.72)]">
                          {metric === 'triggered_count' ? row.triggeredCount : formatPercent(row.value)}
                        </span>
                      </div>
                      <div className="h-1.5 overflow-hidden rounded-full bg-[rgba(15,130,107,0.15)]">
                        <div className="h-full rounded-full bg-[linear-gradient(90deg,#0f826b,#159997)]" style={{ width: `${width}%` }} />
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>

            <p className="m-0 text-xs text-[rgba(18,34,54,0.65)]">
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
