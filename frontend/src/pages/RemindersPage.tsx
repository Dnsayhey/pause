import { useEffect, useRef, useState } from 'react';
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
  createPanelRequestId: number;
  createPanelAnchor: { top: number; right: number } | null;
  onReminderEnabledChange: (id: string, enabled: boolean) => void;
  onReminderIntervalDraftChange: (id: string, value: string) => void;
  onReminderIntervalDraftNormalize: (id: string, value: string, unitSec: number) => number;
  onReminderBreakDraftChange: (id: string, value: string) => void;
  onReminderBreakDraftNormalize: (id: string, value: string, unitSec: number) => number;
  onReminderDraftCommit: (
    id: string,
    intervalValue: string,
    breakValue: string,
    intervalUnitSec: number,
    breakUnitSec: number
  ) => Promise<void> | void;
  onReminderEditCancel: (id: string) => void;
  onReminderDelete: (id: string) => Promise<boolean>;
  onCreateReminder: (
    name: string,
    intervalSec: number,
    breakSec: number,
    reminderType: 'rest' | 'notify'
  ) => Promise<boolean>;
};

type ReminderTodayStat = {
  triggeredCount: number;
  completedCount: number;
};

type IntervalUnit = 'hour' | 'minute';
type BreakUnit = 'minute' | 'second';
type ReminderEditUnits = {
  intervalUnitSec: number;
  breakUnitSec: number;
};

const INTERVAL_SWITCH_UNITS_SEC = [60, 3600] as const;
const BREAK_SWITCH_UNITS_SEC = [60, 1] as const;

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

function parsePositiveInteger(value: string): number | null {
  const parsed = Number.parseInt(value.trim(), 10);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return null;
  }
  return parsed;
}

function isCustomReminder(id: string): boolean {
  return id !== 'eye' && id !== 'stand';
}

function unitBounds(min: number, max: number | undefined, baseUnitSec: number, activeUnitSec: number) {
  const minSec = min * baseUnitSec;
  const maxSec = max === undefined ? undefined : max * baseUnitSec;
  const unitMin = Math.max(1, Math.ceil(minSec / activeUnitSec));
  const unitMax = maxSec === undefined ? undefined : Math.max(unitMin, Math.floor(maxSec / activeUnitSec));
  return { unitMin, unitMax };
}

function clampUnitValue(value: number, min: number, max?: number): number {
  if (value < min) return min;
  if (max !== undefined && value > max) return max;
  return value;
}

function parseAndClampDraft(value: string, fallbackSec: number, unitSec: number, min: number, max?: number): number {
  const parsed = parsePositiveInteger(value);
  const fallback = clampUnitValue(Math.round(Math.max(1, fallbackSec) / unitSec), min, max);
  if (parsed === null) {
    return fallback;
  }
  return clampUnitValue(parsed, min, max);
}

function deriveDefaultEditUnits(reminder: ReminderConfig): ReminderEditUnits {
  const custom = isCustomReminder(reminder.id);
  const intervalUnitSec =
    custom && reminder.intervalSec >= 3600 && reminder.intervalSec % 3600 === 0 ? 3600 : 60;
  const breakUnitSec =
    custom && reminder.breakSec >= 60 && reminder.breakSec % 60 === 0 ? 60 : 1;
  return {
    intervalUnitSec,
    breakUnitSec
  };
}

function customFriendlyBounds(
  reminderID: string,
  min: number,
  max: number | undefined,
  baseUnitSec: number,
  activeUnitSec: number
) {
  if (isCustomReminder(reminderID)) {
    return {
      unitMin: 1,
      unitMax: undefined as number | undefined
    };
  }
  return unitBounds(min, max, baseUnitSec, activeUnitSec);
}

const createFieldInputClassName =
  'h-[34px] w-full rounded-[10px] border border-[var(--dialog-field-border)] bg-[var(--dialog-field-bg)] px-3 text-[14px] text-[var(--dialog-field-text)] outline-none transition-colors focus:border-[var(--dialog-field-focus-border)] focus:ring-2 focus:ring-[var(--control-focus-ring)]';

const createFieldSelectClassName =
  'h-[34px] w-full cursor-pointer rounded-[10px] border border-[var(--dialog-field-border)] bg-[var(--dialog-field-bg)] px-2 text-[14px] text-[var(--dialog-field-text)] outline-none transition-colors focus:border-[var(--dialog-field-focus-border)] focus:ring-2 focus:ring-[var(--control-focus-ring)]';

export function RemindersPage({
  locale,
  reminders,
  runtimeReminders,
  reminderDrafts,
  createPanelRequestId,
  createPanelAnchor,
  onReminderEnabledChange,
  onReminderIntervalDraftChange,
  onReminderIntervalDraftNormalize,
  onReminderBreakDraftChange,
  onReminderBreakDraftNormalize,
  onReminderDraftCommit,
  onReminderEditCancel,
  onReminderDelete,
  onCreateReminder
}: RemindersPageProps) {
  const lastHandledCreatePanelRequestIdRef = useRef(createPanelRequestId);
  const [todayStatsByReminderID, setTodayStatsByReminderID] = useState<Record<string, ReminderTodayStat>>({});
  const [editUnitsByReminderID, setEditUnitsByReminderID] = useState<Record<string, ReminderEditUnits>>({});
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [createName, setCreateName] = useState('');
  const [createType, setCreateType] = useState<'rest' | 'notify'>('rest');
  const [createIntervalValue, setCreateIntervalValue] = useState('25');
  const [createIntervalUnit, setCreateIntervalUnit] = useState<IntervalUnit>('minute');
  const [createBreakValue, setCreateBreakValue] = useState('1');
  const [createBreakUnit, setCreateBreakUnit] = useState<BreakUnit>('minute');
  const [createError, setCreateError] = useState('');

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

  useEffect(() => {
    setEditUnitsByReminderID((prev) => {
      const next: Record<string, ReminderEditUnits> = {};
      for (const reminder of reminders) {
        next[reminder.id] = prev[reminder.id] ?? deriveDefaultEditUnits(reminder);
      }
      return next;
    });
  }, [reminders]);

  useEffect(() => {
    if (!isCreateOpen) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !isCreating) {
        setIsCreateOpen(false);
      }
    };
    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.removeEventListener('keydown', onKeyDown);
    };
  }, [isCreateOpen, isCreating]);

  useEffect(() => {
    if (createPanelRequestId <= 0) return;
    if (createPanelRequestId === lastHandledCreatePanelRequestIdRef.current) return;
    lastHandledCreatePanelRequestIdRef.current = createPanelRequestId;
    const customReminderCount = reminders.filter((reminder) => reminder.id !== 'eye' && reminder.id !== 'stand').length;
    setCreateName(`${t(locale, 'addReminderDefaultName')} ${customReminderCount + 1}`);
    setCreateType('rest');
    setCreateIntervalValue('25');
    setCreateIntervalUnit('minute');
    setCreateBreakValue('1');
    setCreateBreakUnit('minute');
    setCreateError('');
    setIsCreateOpen(true);
  }, [createPanelRequestId, locale, reminders]);

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

  const closeCreatePanel = () => {
    if (isCreating) return;
    setIsCreateOpen(false);
    setCreateError('');
  };

  const submitCreateReminder = async () => {
    const name = createName.trim();
    if (name === '') {
      setCreateError(t(locale, 'addReminderNameRequired'));
      return;
    }
    const parsedInterval = parsePositiveInteger(createIntervalValue);
    const parsedBreak = parsePositiveInteger(createBreakValue);
    const intervalValue = parsedInterval ?? (createIntervalUnit === 'hour' ? 1 : 25);
    const breakValue = parsedBreak ?? 1;
    const intervalSec = intervalValue * (createIntervalUnit === 'hour' ? 3600 : 60);
    const breakSec = createType === 'notify' ? 1 : breakValue * (createBreakUnit === 'minute' ? 60 : 1);
    setCreateIntervalValue(String(intervalValue));
    setCreateBreakValue(String(breakValue));
    setCreateError('');
    setIsCreating(true);
    try {
      const created = await onCreateReminder(name, intervalSec, breakSec, createType);
      if (created) {
        setIsCreateOpen(false);
      }
    } finally {
      setIsCreating(false);
    }
  };

  return (
    <>
      <section className="mt-3 mx-auto grid w-full max-w-[760px] grid-cols-1 gap-1 px-2 sm:px-3">
        {reminders.map((reminder) => {
          const isNotificationReminder = reminder.reminderType === 'notify';
          const isCustom = isCustomReminder(reminder.id);
          const spec = reminderFieldSpecByID(reminder.id);
          const unitState = editUnitsByReminderID[reminder.id] ?? deriveDefaultEditUnits(reminder);
          const intervalBounds = customFriendlyBounds(
            reminder.id,
            spec.intervalMin,
            spec.intervalMax,
            spec.intervalUnitSec,
            unitState.intervalUnitSec
          );
          const breakBounds = customFriendlyBounds(
            reminder.id,
            spec.breakMin,
            spec.breakMax,
            spec.breakUnitSec,
            unitState.breakUnitSec
          );
          const draft = reminderDrafts[reminder.id];
          const intervalRaw = draft?.interval ?? String(toDraftIntervalValue(reminder.intervalSec, spec));
          const breakRaw = draft?.break ?? String(toDraftBreakValue(reminder.breakSec, spec));
          const intervalValue = String(
            parseAndClampDraft(
              intervalRaw,
              reminder.intervalSec,
              unitState.intervalUnitSec,
              intervalBounds.unitMin,
              intervalBounds.unitMax
            )
          );
          const breakValue = String(
            parseAndClampDraft(
              breakRaw,
              reminder.breakSec,
              unitState.breakUnitSec,
              breakBounds.unitMin,
              breakBounds.unitMax
            )
          );
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
              cancelLabel={t(locale, 'addReminderCancel')}
              deleteLabel={t(locale, 'delete')}
              metaText={metaText}
              intervalLabel={t(locale, spec.intervalLabelKey)}
              intervalValue={intervalValue}
              intervalUnitSec={unitState.intervalUnitSec}
              intervalMin={intervalBounds.unitMin}
              intervalMax={intervalBounds.unitMax}
              canToggleIntervalUnit={isCustom}
              onIntervalUnitToggle={() => {
                const currentUnitSec = unitState.intervalUnitSec;
                const currentIndex = INTERVAL_SWITCH_UNITS_SEC.indexOf(currentUnitSec as (typeof INTERVAL_SWITCH_UNITS_SEC)[number]);
                const nextUnitSec =
                  INTERVAL_SWITCH_UNITS_SEC[
                    currentIndex >= 0 ? (currentIndex + 1) % INTERVAL_SWITCH_UNITS_SEC.length : 0
                  ];
                const currentValue = parseAndClampDraft(
                  intervalValue,
                  reminder.intervalSec,
                  currentUnitSec,
                  intervalBounds.unitMin,
                  intervalBounds.unitMax
                );
                const currentSec = currentValue * currentUnitSec;
                const nextBounds = customFriendlyBounds(
                  reminder.id,
                  spec.intervalMin,
                  spec.intervalMax,
                  spec.intervalUnitSec,
                  nextUnitSec
                );
                const nextValue = parseAndClampDraft('', currentSec, nextUnitSec, nextBounds.unitMin, nextBounds.unitMax);
                setEditUnitsByReminderID((prev) => ({
                  ...prev,
                  [reminder.id]: {
                    ...unitState,
                    intervalUnitSec: nextUnitSec
                  }
                }));
                onReminderIntervalDraftChange(reminder.id, String(nextValue));
              }}
              onIntervalChange={(value) => {
                onReminderIntervalDraftChange(reminder.id, value);
              }}
              onIntervalNormalize={(value) => {
                onReminderIntervalDraftNormalize(reminder.id, value, unitState.intervalUnitSec);
              }}
              breakLabel={t(locale, spec.breakLabelKey)}
              breakValue={breakValue}
              breakUnitSec={unitState.breakUnitSec}
              breakMin={breakBounds.unitMin}
              breakMax={breakBounds.unitMax}
              canToggleBreakUnit={isCustom && !isNotificationReminder}
              onBreakUnitToggle={() => {
                const currentUnitSec = unitState.breakUnitSec;
                const currentIndex = BREAK_SWITCH_UNITS_SEC.indexOf(currentUnitSec as (typeof BREAK_SWITCH_UNITS_SEC)[number]);
                const nextUnitSec =
                  BREAK_SWITCH_UNITS_SEC[currentIndex >= 0 ? (currentIndex + 1) % BREAK_SWITCH_UNITS_SEC.length : 0];
                const currentValue = parseAndClampDraft(
                  breakValue,
                  reminder.breakSec,
                  currentUnitSec,
                  breakBounds.unitMin,
                  breakBounds.unitMax
                );
                const currentSec = currentValue * currentUnitSec;
                const nextBounds = customFriendlyBounds(
                  reminder.id,
                  spec.breakMin,
                  spec.breakMax,
                  spec.breakUnitSec,
                  nextUnitSec
                );
                const nextValue = parseAndClampDraft('', currentSec, nextUnitSec, nextBounds.unitMin, nextBounds.unitMax);
                setEditUnitsByReminderID((prev) => ({
                  ...prev,
                  [reminder.id]: {
                    ...unitState,
                    breakUnitSec: nextUnitSec
                  }
                }));
                onReminderBreakDraftChange(reminder.id, String(nextValue));
              }}
              onBreakChange={(value) => {
                onReminderBreakDraftChange(reminder.id, value);
              }}
              onBreakNormalize={(value) => {
                onReminderBreakDraftNormalize(reminder.id, value, unitState.breakUnitSec);
              }}
              onDoneEdit={() =>
                onReminderDraftCommit(
                  reminder.id,
                  intervalValue,
                  breakValue,
                  unitState.intervalUnitSec,
                  unitState.breakUnitSec
                )
              }
              onCancelEdit={() => {
                onReminderEditCancel(reminder.id);
                setEditUnitsByReminderID((prev) => ({
                  ...prev,
                  [reminder.id]: deriveDefaultEditUnits(reminder)
                }));
              }}
              onDelete={() => onReminderDelete(reminder.id).then(() => undefined)}
            />
          );
        })}
      </section>

      {isCreateOpen ? (
        <>
          <button
            type="button"
            aria-label={t(locale, 'addReminderCancel')}
            className="fixed inset-[-8px] z-30 bg-[var(--dialog-scrim)]"
            onClick={closeCreatePanel}
          />
          <section
            role="dialog"
            aria-modal="true"
            aria-label={t(locale, 'addReminderCardTitle')}
            className="fixed z-40 w-[340px] max-w-[calc(100vw-1.5rem)] rounded-[14px] border border-[var(--surface-border-strong)] bg-[var(--surface-bg)] p-4 shadow-[var(--surface-shadow)] max-sm:left-3 max-sm:right-3 max-sm:w-auto"
            style={{
              top: `${createPanelAnchor?.top ?? 76}px`,
              right: `${createPanelAnchor?.right ?? 20}px`
            }}
          >
            <div className="grid grid-cols-[1fr_110px] gap-2 max-sm:grid-cols-1">
              <label className="text-xs text-[var(--text-secondary)]">
                {t(locale, 'addReminderNameLabel')}
                <input
                  autoFocus
                  value={createName}
                  onChange={(event) => setCreateName(event.target.value)}
                  placeholder={t(locale, 'addReminderNamePlaceholder')}
                  className={`mt-1 ${createFieldInputClassName}`}
                />
              </label>
              <label className="text-xs text-[var(--text-secondary)]">
                {t(locale, 'addReminderTypeLabel')}
                <select
                  value={createType}
                  onChange={(event) => setCreateType(event.target.value as 'rest' | 'notify')}
                  className={`mt-1 ${createFieldSelectClassName}`}
                >
                  <option value="rest">{t(locale, 'addReminderTypeRest')}</option>
                  <option value="notify">{t(locale, 'addReminderTypeNotify')}</option>
                </select>
              </label>
            </div>

            <div className={`mt-3 grid gap-2 ${createType === 'notify' ? 'grid-cols-1' : 'grid-cols-2'} max-sm:grid-cols-1`}>
              <label className="text-xs text-[var(--text-secondary)]">
                {t(locale, 'addReminderIntervalLabel')}
                <div className="mt-1 flex gap-2">
                  <input
                    type="number"
                    min={1}
                    step={1}
                    value={createIntervalValue}
                    onChange={(event) => setCreateIntervalValue(event.target.value)}
                    className={createFieldInputClassName}
                  />
                  <select
                    value={createIntervalUnit}
                    onChange={(event) => setCreateIntervalUnit(event.target.value as IntervalUnit)}
                    className={`${createFieldSelectClassName} max-w-[88px]`}
                  >
                    <option value="hour">{t(locale, 'addReminderUnitHour')}</option>
                    <option value="minute">{t(locale, 'addReminderUnitMinute')}</option>
                  </select>
                </div>
              </label>

              {createType === 'rest' ? (
                <label className="text-xs text-[var(--text-secondary)]">
                  {t(locale, 'addReminderBreakLabel')}
                  <div className="mt-1 flex gap-2">
                    <input
                      type="number"
                      min={1}
                      step={1}
                      value={createBreakValue}
                      onChange={(event) => setCreateBreakValue(event.target.value)}
                      className={createFieldInputClassName}
                    />
                    <select
                      value={createBreakUnit}
                      onChange={(event) => setCreateBreakUnit(event.target.value as BreakUnit)}
                      className={`${createFieldSelectClassName} max-w-[88px]`}
                    >
                      <option value="minute">{t(locale, 'addReminderUnitMinute')}</option>
                      <option value="second">{t(locale, 'addReminderUnitSecond')}</option>
                    </select>
                  </div>
                </label>
              ) : null}
            </div>

            {createError ? <p className="m-0 mt-2 text-xs text-[var(--error-text)]">{createError}</p> : null}

            <div className="mt-4 flex items-center justify-end gap-2">
              <button
                type="button"
                onClick={closeCreatePanel}
                disabled={isCreating}
                className="inline-flex min-h-[var(--control-height)] cursor-pointer items-center justify-center rounded-[11px] border border-transparent bg-[var(--dialog-subtle-btn-bg)] px-[12px] py-1 text-[13px] text-[var(--dialog-subtle-btn-text)] transition-colors hover:text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--control-focus-ring)] disabled:cursor-not-allowed disabled:opacity-60"
              >
                {t(locale, 'addReminderCancel')}
              </button>
              <button
                type="button"
                onClick={() => {
                  void submitCreateReminder();
                }}
                disabled={isCreating}
                className="inline-flex min-h-[var(--control-height)] cursor-pointer items-center justify-center rounded-[11px] border border-transparent bg-[linear-gradient(140deg,var(--seg-active),var(--seg-active-strong))] px-[14px] py-1 text-[13px] font-medium text-white transition-opacity hover:opacity-95 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--control-focus-ring)] disabled:cursor-not-allowed disabled:opacity-60"
              >
                {isCreating ? t(locale, 'addReminderCreating') : t(locale, 'addReminderCreate')}
              </button>
            </div>
          </section>
        </>
      ) : null}
    </>
  );
}
