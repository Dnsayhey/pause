import { useEffect, useState } from 'react';
import { getAnalyticsWeeklyStats } from '../api';
import { localizeReason, t, type Locale } from '../i18n';
import { reminderFieldSpecByID, toDraftBreakValue, toDraftIntervalValue } from '../reminderFields';
import type { ReminderConfig, ReminderRuntime } from '../types';
import { ReminderCard } from '../components/ReminderCard';

type ReminderDrafts = Record<string, { interval: string; break: string }>;

type RemindersPageProps = {
  locale: Locale;
  reminders: ReminderConfig[];
  runtimeReminders: ReminderRuntime[];
  reminderDrafts: ReminderDrafts;
  onReminderEnabledChange: (id: string, enabled: boolean) => void;
  onReminderIntervalDraftChange: (id: string, value: string) => void;
  onReminderIntervalDraftNormalize: (id: string, value: string) => number;
  onReminderBreakDraftChange: (id: string, value: string) => void;
  onReminderBreakDraftNormalize: (id: string, value: string) => number;
  onReminderDraftCommit: (id: string, intervalValue: string, breakValue: string) => Promise<void> | void;
  onReminderEditCancel: (id: string) => void;
};

type ReminderTodayStat = {
  triggeredCount: number;
  completedCount: number;
};

function reminderTitle(reminder: ReminderConfig, locale: Locale): string {
  const id = reminder.id;
  if (id === 'eye') {
    return t(locale, 'eyeReminder');
  }
  if (id === 'stand') {
    return t(locale, 'standReminder');
  }
  if (reminder.name.trim() !== '') {
    return reminder.name;
  }
  return localizeReason(id, locale);
}

export function RemindersPage({
  locale,
  reminders,
  runtimeReminders,
  reminderDrafts,
  onReminderEnabledChange,
  onReminderIntervalDraftChange,
  onReminderIntervalDraftNormalize,
  onReminderBreakDraftChange,
  onReminderBreakDraftNormalize,
  onReminderDraftCommit,
  onReminderEditCancel
}: RemindersPageProps) {
  const [todayStatsByReminderID, setTodayStatsByReminderID] = useState<Record<string, ReminderTodayStat>>({});

  useEffect(() => {
    let cancelled = false;
    let timer: number | null = null;

    const loadTodayStats = async () => {
      try {
        const now = new Date();
        const startOfDay = new Date(now);
        startOfDay.setHours(0, 0, 0, 0);
        const fromSec = Math.floor(startOfDay.getTime() / 1000);
        const toSec = Math.floor(now.getTime() / 1000);
        const weekly = await getAnalyticsWeeklyStats(fromSec, toSec);
        if (cancelled) return;
        const nextMap: Record<string, ReminderTodayStat> = {};
        for (const item of weekly.reminders) {
          nextMap[item.reminderId] = {
            triggeredCount: item.triggeredCount,
            completedCount: item.completedCount
          };
        }
        setTodayStatsByReminderID(nextMap);
      } catch {
        if (!cancelled) {
          setTodayStatsByReminderID({});
        }
      }
    };

    void loadTodayStats();
    timer = window.setInterval(() => {
      void loadTodayStats();
    }, 60_000);

    return () => {
      cancelled = true;
      if (timer !== null) {
        window.clearInterval(timer);
      }
    };
  }, []);

  const formatTodayRate = (stat?: ReminderTodayStat): string => {
    if (!stat || stat.triggeredCount <= 0) return '--';
    return `${Math.round((stat.completedCount / stat.triggeredCount) * 100)}%`;
  };

  const formatTodayDone = (stat?: ReminderTodayStat): string => {
    if (!stat || stat.triggeredCount <= 0) return '--';
    return `${stat.completedCount}/${stat.triggeredCount}`;
  };

  const formatNextBreak = (runtime?: ReminderRuntime): string => {
    if (!runtime || !runtime.enabled || runtime.paused) {
      return t(locale, 'nextBreakOff');
    }
    const sec = Math.max(0, Math.floor(runtime.nextInSec));
    const target = new Date(Date.now() + sec * 1000);
    const hours = String(target.getHours()).padStart(2, '0');
    const minutes = String(target.getMinutes()).padStart(2, '0');
    return `${hours}:${minutes}`;
  };

  return (
    <section className="mt-3 mx-auto grid w-full max-w-[760px] grid-cols-1 gap-1 px-2 sm:px-3">
      {reminders.map((reminder) => {
        const isNotificationReminder = reminder.reminderType === 'notify';
        const spec = reminderFieldSpecByID(reminder.id);
        const draft = reminderDrafts[reminder.id];
        const intervalValue = draft?.interval ?? String(toDraftIntervalValue(reminder.intervalSec, spec));
        const breakValue = draft?.break ?? String(toDraftBreakValue(reminder.breakSec, spec));
        const todayStat = todayStatsByReminderID[reminder.id];
        const runtimeReminder = runtimeReminders.find((item) => item.id === reminder.id);
        const metaText = isNotificationReminder
          ? `${t(locale, 'reminderMetaNextNotify')} ${formatNextBreak(runtimeReminder)}`
          : `${t(locale, 'reminderMetaNextBreak')} ${formatNextBreak(runtimeReminder)} · ${t(
              locale,
              'reminderMetaTodayRate'
            )} ${formatTodayRate(todayStat)} - ${formatTodayDone(todayStat)}`;

        return (
          <ReminderCard
            key={reminder.id}
            locale={locale}
            variant={isNotificationReminder ? 'notify' : 'rest'}
            title={reminderTitle(reminder, locale)}
            enabledLabel={t(locale, 'enabled')}
            enabled={reminder.enabled}
            onEnabledChange={(checked) => {
              onReminderEnabledChange(reminder.id, checked);
            }}
            editLabel={t(locale, 'edit')}
            doneLabel={t(locale, 'done')}
            metaText={metaText}
            intervalLabel={t(locale, spec.intervalLabelKey)}
            intervalValue={intervalValue}
            intervalUnitSec={spec.intervalUnitSec}
            intervalMin={spec.intervalMin}
            intervalMax={spec.intervalMax}
            onIntervalChange={(value) => {
              onReminderIntervalDraftChange(reminder.id, value);
            }}
            onIntervalNormalize={(value) => {
              onReminderIntervalDraftNormalize(reminder.id, value);
            }}
            breakLabel={t(locale, spec.breakLabelKey)}
            breakValue={breakValue}
            breakUnitSec={spec.breakUnitSec}
            breakMin={spec.breakMin}
            breakMax={spec.breakMax}
            onBreakChange={(value) => {
              onReminderBreakDraftChange(reminder.id, value);
            }}
            onBreakNormalize={(value) => {
              onReminderBreakDraftNormalize(reminder.id, value);
            }}
            onDoneEdit={() => onReminderDraftCommit(reminder.id, intervalValue, breakValue)}
            onCancelEdit={() => {
              onReminderEditCancel(reminder.id);
            }}
          />
        );
      })}
    </section>
  );
}
