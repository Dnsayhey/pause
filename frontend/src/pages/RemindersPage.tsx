import { useEffect, useMemo, useState, type KeyboardEvent } from 'react';
import { getAnalyticsWeeklyStats } from '../api';
import { localizeReason, t, type Locale } from '../i18n';
import { reminderFieldSpecByID, toDraftBreakValue, toDraftIntervalValue } from '../reminderFields';
import type { ReminderConfig, ReminderRuntime } from '../types';
import { ReminderCard } from '../components/ReminderCard';
import { GlassCard } from '../components/ui';

type ReminderDrafts = Record<string, { interval: string; break: string }>;

type AddReminderInput = {
  name: string;
  intervalSec: number;
  breakSec: number;
  reminderType: 'rest' | 'notify';
};

type RemindersPageProps = {
  locale: Locale;
  reminders: ReminderConfig[];
  runtimeReminders: ReminderRuntime[];
  reminderDrafts: ReminderDrafts;
  onAddReminder: (input: AddReminderInput) => Promise<boolean> | boolean;
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

const inlineNumberInputClassName =
  'number-input w-[3.6ch] min-w-[3.6ch] cursor-text appearance-none border-0 border-b border-[var(--surface-border-strong)] bg-transparent px-0 py-[1px] text-right text-[15px] leading-[1.2] font-medium text-[var(--text-primary)] caret-[var(--accent)] shadow-none outline-none transition-colors duration-150 hover:border-[var(--field-border-muted)] focus:border-[var(--accent)]';

const nameInputClassName =
  'w-full min-w-0 border-0 border-b border-[var(--surface-border-strong)] bg-transparent px-0 py-[1px] text-[18px] leading-[1.2] font-medium text-[var(--text-primary)] caret-[var(--accent)] shadow-none outline-none transition-colors duration-150 placeholder:text-[var(--text-tertiary)] hover:border-[var(--field-border-muted)] focus:border-[var(--accent)]';

function parsePositiveInteger(value: string): number | null {
  const numeric = Number(value);
  if (!Number.isFinite(numeric)) return null;
  const rounded = Math.round(numeric);
  return rounded > 0 ? rounded : null;
}

function clampPositiveInteger(value: string, fallback: number, min: number, max?: number): number {
  const parsed = parsePositiveInteger(value);
  if (parsed === null) return fallback;
  if (parsed < min) return min;
  if (max !== undefined && parsed > max) return max;
  return parsed;
}

function suppressEnterBlur(event: KeyboardEvent<HTMLInputElement>) {
  if (event.key !== 'Enter') return;
  event.preventDefault();
  event.currentTarget.blur();
}

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
  onAddReminder,
  onReminderEnabledChange,
  onReminderIntervalDraftChange,
  onReminderIntervalDraftNormalize,
  onReminderBreakDraftChange,
  onReminderBreakDraftNormalize,
  onReminderDraftCommit,
  onReminderEditCancel
}: RemindersPageProps) {
  const [todayStatsByReminderID, setTodayStatsByReminderID] = useState<Record<string, ReminderTodayStat>>({});
  const customReminderCount = useMemo(
    () => reminders.filter((reminder) => reminder.id !== 'eye' && reminder.id !== 'stand').length,
    [reminders]
  );
  const nextDefaultName = useMemo(() => {
    const baseName = t(locale, 'addReminderDefaultName');
    return locale === 'zh-CN' ? `${baseName}${customReminderCount + 1}` : `${baseName} ${customReminderCount + 1}`;
  }, [customReminderCount, locale]);
  const [isComposerOpen, setIsComposerOpen] = useState(false);
  const [isCreatingReminder, setIsCreatingReminder] = useState(false);
  const [newReminderKind, setNewReminderKind] = useState<'rest' | 'notify'>('rest');
  const [newReminderName, setNewReminderName] = useState(nextDefaultName);
  const [newIntervalMin, setNewIntervalMin] = useState('60');
  const [newBreakMin, setNewBreakMin] = useState('5');
  const [composerError, setComposerError] = useState('');

  useEffect(() => {
    if (!isComposerOpen) {
      setNewReminderName(nextDefaultName);
    }
  }, [isComposerOpen, nextDefaultName]);

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

  const handleOpenComposer = () => {
    if (isCreatingReminder) return;
    setComposerError('');
    setNewReminderKind('rest');
    setNewReminderName(nextDefaultName);
    setNewIntervalMin('60');
    setNewBreakMin('5');
    setIsComposerOpen(true);
  };

  const handleCancelComposer = () => {
    if (isCreatingReminder) return;
    setIsComposerOpen(false);
    setComposerError('');
    setNewReminderKind('rest');
    setNewReminderName(nextDefaultName);
    setNewIntervalMin('60');
    setNewBreakMin('5');
  };

  const normalizeNewInterval = (value: string) => {
    const nextValue = clampPositiveInteger(value, 60, 1, 24 * 60);
    setNewIntervalMin(String(nextValue));
    return nextValue;
  };

  const normalizeNewBreak = (value: string) => {
    const nextValue = clampPositiveInteger(value, 5, 1, 120);
    setNewBreakMin(String(nextValue));
    return nextValue;
  };

  const handleCreateReminder = async () => {
    if (isCreatingReminder) return;
    const name = newReminderName.trim();
    if (name === '') {
      setComposerError(t(locale, 'addReminderNameRequired'));
      return;
    }
    const intervalMin = normalizeNewInterval(newIntervalMin);
    const breakMin = normalizeNewBreak(newBreakMin);
    setComposerError('');

    setIsCreatingReminder(true);
    try {
      const reminderType = newReminderKind;
      const created = await onAddReminder({
        name,
        intervalSec: intervalMin * 60,
        breakSec: reminderType === 'notify' ? 20 : breakMin * 60,
        reminderType
      });
      if (created === false) {
        return;
      }
      setIsComposerOpen(false);
      setNewReminderKind('rest');
      setNewReminderName(nextDefaultName);
      setNewIntervalMin('60');
      setNewBreakMin('5');
    } finally {
      setIsCreatingReminder(false);
    }
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
      {!isComposerOpen ? (
        <button type="button" className="w-full border-0 bg-transparent p-0 text-left" onClick={handleOpenComposer}>
          <GlassCard
            as="article"
            className="group/reminder min-h-[126px] border border-[var(--surface-border)] bg-[var(--app-bg)] shadow-none transition-colors hover:border-[var(--surface-border-strong)]"
          >
            <div className="flex min-h-[90px] items-center justify-center">
              <span className="inline-flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm font-medium text-[var(--text-secondary)] transition-colors group-hover/reminder:bg-[var(--control-hover-bg)] group-hover/reminder:text-[var(--text-primary)]">
                <svg aria-hidden="true" viewBox="0 0 16 16" className="h-3.5 w-3.5" fill="none">
                  <path d="M8 3v10M3 8h10" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
                </svg>
                {t(locale, 'addReminder')}
              </span>
            </div>
          </GlassCard>
        </button>
      ) : (
        <GlassCard
          as="article"
          className="group/reminder min-h-[126px] border border-[var(--surface-border)] bg-[var(--app-bg)] shadow-none transition-colors hover:border-[var(--surface-border-strong)]"
        >
          <>
            <div className="mb-3 flex items-start justify-between gap-3">
              <div className="flex min-w-0 flex-1 items-center gap-2.5">
                <span aria-hidden="true" className="h-4 w-[3px] rounded-full bg-[var(--text-primary)] opacity-75 transition-colors" />
                <input
                  type="text"
                  aria-label={t(locale, 'addReminderNameLabel')}
                  placeholder={t(locale, 'addReminderNamePlaceholder')}
                  value={newReminderName}
                  className={nameInputClassName}
                  onChange={(event) => {
                    setNewReminderName(event.target.value);
                    if (composerError !== '') {
                      setComposerError('');
                    }
                  }}
                  onKeyDown={(event) => {
                    if (event.key !== 'Enter') return;
                    event.preventDefault();
                    void handleCreateReminder();
                  }}
                />
              </div>
              <div className="flex items-center gap-1">
                <button
                  type="button"
                  className="inline-flex h-7 items-center rounded-md border-0 bg-transparent px-2 text-xs text-[var(--text-secondary)] transition-colors hover:bg-[var(--control-hover-bg)] hover:text-[var(--text-primary)] focus-visible:outline-none"
                  onClick={handleCancelComposer}
                  disabled={isCreatingReminder}
                >
                  {t(locale, 'addReminderCancel')}
                </button>
                <button
                  type="button"
                  className="inline-flex h-7 items-center rounded-md border-0 bg-[var(--text-primary)] px-2.5 text-xs font-medium text-[var(--surface-bg)] transition-opacity focus-visible:outline-none disabled:opacity-60"
                  onClick={() => {
                    void handleCreateReminder();
                  }}
                  disabled={isCreatingReminder}
                >
                  {isCreatingReminder ? t(locale, 'addReminderCreating') : t(locale, 'addReminderCreate')}
                </button>
              </div>
            </div>
            <div className="mb-3 inline-flex rounded-full border border-[var(--seg-border)] bg-[var(--seg-bg)] p-1 shadow-[0_1px_1px_rgba(0,0,0,0.06)]">
              <button
                type="button"
                className={`rounded-full px-3 py-1 text-xs font-medium transition-colors ${
                  newReminderKind === 'rest'
                    ? 'bg-[linear-gradient(140deg,var(--seg-active),var(--seg-active-strong))] text-white shadow-[0_1px_2px_rgba(0,0,0,0.18)]'
                    : 'text-[var(--seg-text)] hover:bg-[var(--seg-hover-bg)] hover:text-[var(--text-primary)]'
                }`}
                onClick={() => {
                  setNewReminderKind('rest');
                }}
                disabled={isCreatingReminder}
              >
                {t(locale, 'addReminderTypeRest')}
              </button>
              <button
                type="button"
                className={`rounded-full px-3 py-1 text-xs font-medium transition-colors ${
                  newReminderKind === 'notify'
                    ? 'bg-[linear-gradient(140deg,var(--seg-active),var(--seg-active-strong))] text-white shadow-[0_1px_2px_rgba(0,0,0,0.18)]'
                    : 'text-[var(--seg-text)] hover:bg-[var(--seg-hover-bg)] hover:text-[var(--text-primary)]'
                }`}
                onClick={() => {
                  setNewReminderKind('notify');
                }}
                disabled={isCreatingReminder}
              >
                {t(locale, 'addReminderTypeNotify')}
              </button>
            </div>
            <div className="flex flex-wrap items-center gap-x-1.5 gap-y-1 text-[15px] leading-[1.45] text-[var(--text-primary)]">
              {newReminderKind === 'notify' ? (
                locale === 'zh-CN' ? (
                  <>
                    <span>每隔</span>
                    <input
                      type="number"
                      min={1}
                      max={24 * 60}
                      step={1}
                      aria-label={t(locale, 'addReminderIntervalMin')}
                      className={inlineNumberInputClassName}
                      value={newIntervalMin}
                      onChange={(event) => {
                        setNewIntervalMin(event.target.value);
                      }}
                      onKeyDown={suppressEnterBlur}
                      onBlur={(event) => {
                        normalizeNewInterval(event.currentTarget.value);
                      }}
                    />
                    <span>{t(locale, 'addReminderIntervalMin')}</span>
                    <span>提醒一次</span>
                  </>
                ) : (
                  <>
                    <span>Notify every</span>
                    <input
                      type="number"
                      min={1}
                      max={24 * 60}
                      step={1}
                      aria-label={t(locale, 'addReminderIntervalMin')}
                      className={inlineNumberInputClassName}
                      value={newIntervalMin}
                      onChange={(event) => {
                        setNewIntervalMin(event.target.value);
                      }}
                      onKeyDown={suppressEnterBlur}
                      onBlur={(event) => {
                        normalizeNewInterval(event.currentTarget.value);
                      }}
                    />
                    <span>{t(locale, 'addReminderIntervalMin')}</span>
                  </>
                )
              ) : locale === 'zh-CN' ? (
                <>
                  <span>每隔</span>
                  <input
                    type="number"
                    min={1}
                    max={24 * 60}
                    step={1}
                    aria-label={t(locale, 'addReminderIntervalMin')}
                    className={inlineNumberInputClassName}
                    value={newIntervalMin}
                    onChange={(event) => {
                      setNewIntervalMin(event.target.value);
                    }}
                    onKeyDown={suppressEnterBlur}
                    onBlur={(event) => {
                      normalizeNewInterval(event.currentTarget.value);
                    }}
                  />
                  <span>{t(locale, 'addReminderIntervalMin')}</span>
                  <span>休息</span>
                  <input
                    type="number"
                    min={1}
                    max={120}
                    step={1}
                    aria-label={t(locale, 'addReminderBreakMin')}
                    className={inlineNumberInputClassName}
                    value={newBreakMin}
                    onChange={(event) => {
                      setNewBreakMin(event.target.value);
                    }}
                    onKeyDown={suppressEnterBlur}
                    onBlur={(event) => {
                      normalizeNewBreak(event.currentTarget.value);
                    }}
                  />
                  <span>{t(locale, 'addReminderBreakMin')}</span>
                </>
              ) : (
                <>
                  <span>Every</span>
                  <input
                    type="number"
                    min={1}
                    max={24 * 60}
                    step={1}
                    aria-label={t(locale, 'addReminderIntervalMin')}
                    className={inlineNumberInputClassName}
                    value={newIntervalMin}
                    onChange={(event) => {
                      setNewIntervalMin(event.target.value);
                    }}
                    onKeyDown={suppressEnterBlur}
                    onBlur={(event) => {
                      normalizeNewInterval(event.currentTarget.value);
                    }}
                  />
                  <span>{t(locale, 'addReminderIntervalMin')}</span>
                  <span>take a break for</span>
                  <input
                    type="number"
                    min={1}
                    max={120}
                    step={1}
                    aria-label={t(locale, 'addReminderBreakMin')}
                    className={inlineNumberInputClassName}
                    value={newBreakMin}
                    onChange={(event) => {
                      setNewBreakMin(event.target.value);
                    }}
                    onKeyDown={suppressEnterBlur}
                    onBlur={(event) => {
                      normalizeNewBreak(event.currentTarget.value);
                    }}
                  />
                  <span>{t(locale, 'addReminderBreakMin')}</span>
                </>
              )}
            </div>
            {composerError ? <p className="mt-2 m-0 text-xs leading-[1.35] text-[var(--error-text)]">{composerError}</p> : null}
          </>
        </GlassCard>
      )}
    </section>
  );
}
