import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  createReminder as createReminderAPI,
  deleteReminder as deleteReminderAPI,
  getLaunchAtLogin,
  getReminders,
  getSettings,
  setLaunchAtLogin,
  updateReminders,
  updateSettings
} from '../api';
import {
  isReminderValueValid,
  reminderFieldSpecByID,
} from '../reminderFields';
import type { ReminderConfig, ReminderPatch, Settings, SettingsPatch } from '../types';

const IDLE_THRESHOLD_OPTIONS = [60, 300, 600, 1800, 3600, 7200] as const;
const SOUND_VOLUME_OPTIONS = [20, 40, 60, 80, 100] as const;

type UseSettingsOptions = {
  setError: (message: string) => void;
  refreshRuntime: () => Promise<unknown>;
};

type ReminderDraft = {
  interval: string;
  break: string;
};

function parseInteger(text: string): number | null {
  const trimmed = text.trim();
  if (trimmed === '') return null;
  const value = Number.parseInt(trimmed, 10);
  if (Number.isNaN(value)) return null;
  return value;
}

function deriveUnitBounds(min: number, max: number | undefined, baseUnitSec: number, activeUnitSec: number) {
  const minSec = min * baseUnitSec;
  const maxSec = max === undefined ? undefined : max * baseUnitSec;
  const unitMin = Math.max(1, Math.ceil(minSec / activeUnitSec));
  const unitMax =
    maxSec === undefined ? undefined : Math.max(unitMin, Math.floor(maxSec / activeUnitSec));
  return { unitMin, unitMax, minSec };
}

function deriveCustomDraftUnitSec(totalSec: number, primaryUnitSec: number, secondaryUnitSec: number): number {
  if (totalSec >= primaryUnitSec && totalSec % primaryUnitSec === 0) {
    return primaryUnitSec;
  }
  return secondaryUnitSec;
}

function clampReminderDraftValue(value: number, min: number, max?: number): number {
  if (value < min) return min;
  if (max !== undefined && value > max) return max;
  return value;
}

function nearestOptionValue(value: number, options: readonly number[]): number {
  let nearest = options[0];
  let minDiff = Math.abs(value - nearest);
  for (let i = 1; i < options.length; i += 1) {
    const candidate = options[i];
    const diff = Math.abs(value - candidate);
    if (diff < minDiff) {
      minDiff = diff;
      nearest = candidate;
    }
  }
  return nearest;
}

function reminderByID(reminders: ReminderConfig[], id: number) {
  return reminders.find((reminder) => reminder.id === id);
}

function normalizeReminderName(name: string): string {
  return name.trim();
}

function isValidReminderType(value: unknown): value is 'rest' | 'notify' {
  return value === 'rest' || value === 'notify';
}

function isPositiveInt(value: unknown): value is number {
  return Number.isInteger(value) && Number(value) > 0;
}

function hasNameConflict(reminders: ReminderConfig[], name: string, excludeID?: number): boolean {
  const expected = normalizeReminderName(name).toLowerCase();
  if (expected === '') {
    return false;
  }
  return reminders.some((reminder) => {
    if (excludeID !== undefined && reminder.id === excludeID) {
      return false;
    }
    return normalizeReminderName(reminder.name).toLowerCase() === expected;
  });
}

export function useSettings({ setError, refreshRuntime }: UseSettingsOptions) {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [reminders, setReminders] = useState<ReminderConfig[]>([]);
  const [reminderDrafts, setReminderDrafts] = useState<Record<number, ReminderDraft>>({});
  const [launchAtLogin, setLaunchAtLoginState] = useState(false);

  useEffect(() => {
    let mounted = true;

    const loadSettings = async () => {
      try {
        const [next, reminderRows] = await Promise.all([getSettings(), getReminders()]);
        if (!mounted) return;
        setSettings(next);
        setReminders(reminderRows);

        try {
          const startupState = await getLaunchAtLogin();
          if (mounted) {
            setLaunchAtLoginState(startupState);
          }
        } catch (err) {
          if (mounted) {
            setError(String(err));
          }
        }
      } catch (err) {
        if (mounted) {
          setError(String(err));
        }
      }
    };

    void loadSettings();
    return () => {
      mounted = false;
    };
  }, [setError]);

  const refreshLaunchAtLoginState = useCallback(
    async (silent = false) => {
      try {
        const actual = await getLaunchAtLogin();
        setLaunchAtLoginState(actual);
      } catch (err) {
        if (!silent) {
          setError(String(err));
        }
      }
    },
    [setError]
  );

  useEffect(() => {
    const refreshWhenVisible = () => {
      if (document.visibilityState !== 'visible') return;
      void refreshLaunchAtLoginState(true);
    };

    window.addEventListener('focus', refreshWhenVisible);
    document.addEventListener('visibilitychange', refreshWhenVisible);

    return () => {
      window.removeEventListener('focus', refreshWhenVisible);
      document.removeEventListener('visibilitychange', refreshWhenVisible);
    };
  }, [refreshLaunchAtLoginState]);

  useEffect(() => {
    const nextDrafts: Record<number, ReminderDraft> = {};
    for (const reminder of reminders) {
      const intervalUnitSec = deriveCustomDraftUnitSec(reminder.intervalSec, 3600, 60);
      const breakUnitSec = deriveCustomDraftUnitSec(reminder.breakSec, 60, 1);
      nextDrafts[reminder.id] = {
        interval: String(Math.max(1, Math.round(reminder.intervalSec / intervalUnitSec))),
        break: String(Math.max(1, Math.round(reminder.breakSec / breakUnitSec)))
      };
    }
    setReminderDrafts(nextDrafts);
  }, [reminders]);

  const applyPatch = useCallback(
    async (patch: SettingsPatch) => {
      if (!settings) return;
      setError('');
      try {
        const next = await updateSettings(patch);
        setSettings(next);
        await refreshRuntime();
      } catch (err) {
        setError(String(err));
      }
    },
    [refreshRuntime, setError, settings]
  );

  const applyLaunchAtLogin = useCallback(
    async (enabled: boolean) => {
      setError('');
      try {
        const actual = await setLaunchAtLogin(enabled);
        setLaunchAtLoginState(actual);
      } catch (err) {
        setError(String(err));
        // Keep toggle state synced with real system state when mutation fails.
        void refreshLaunchAtLoginState(true);
      }
    },
    [refreshLaunchAtLoginState, setError]
  );

  const applyReminderPatch = useCallback(
    async (id: number, patch: Omit<ReminderPatch, 'id'>) => {
      if (!isPositiveInt(id)) {
        setError('reminder id is required');
        return;
      }
      const nextPatch: Omit<ReminderPatch, 'id'> = { ...patch };
      if (nextPatch.name !== undefined) {
        const nextName = normalizeReminderName(nextPatch.name);
        if (nextName === '') {
          setError('reminder name is required');
          return;
        }
        if (hasNameConflict(reminders, nextName, id)) {
          setError('reminder already exists');
          return;
        }
        nextPatch.name = nextName;
      }
      if (nextPatch.intervalSec !== undefined && !isPositiveInt(nextPatch.intervalSec)) {
        setError('reminder intervalSec must be > 0');
        return;
      }
      if (nextPatch.breakSec !== undefined && !isPositiveInt(nextPatch.breakSec)) {
        setError('reminder breakSec must be > 0');
        return;
      }
      if (nextPatch.reminderType !== undefined && !isValidReminderType(nextPatch.reminderType)) {
        setError('reminder reminderType must be rest or notify');
        return;
      }

      setError('');
      try {
        const next = await updateReminders([
          {
            id,
            ...nextPatch
          }
        ]);
        setReminders(next);
        await refreshRuntime();
      } catch (err) {
        setError(String(err));
      }
    },
    [refreshRuntime, reminders, setError]
  );

  const createReminder = useCallback(
    async (name: string, intervalSec: number, breakSec: number, reminderType: 'rest' | 'notify'): Promise<boolean> => {
      const nextName = normalizeReminderName(name);
      if (nextName === '') {
        setError('reminder name is required');
        return false;
      }
      if (hasNameConflict(reminders, nextName)) {
        setError('reminder already exists');
        return false;
      }
      if (!isPositiveInt(intervalSec)) {
        setError('reminder intervalSec must be > 0');
        return false;
      }
      if (!isPositiveInt(breakSec)) {
        setError('reminder breakSec must be > 0');
        return false;
      }
      if (!isValidReminderType(reminderType)) {
        setError('reminder reminderType must be rest or notify');
        return false;
      }

      setError('');
      try {
        const next = await createReminderAPI({
          name: nextName,
          intervalSec,
          breakSec,
          enabled: true,
          reminderType
        });
        setReminders(next);
        await refreshRuntime();
        return true;
      } catch (err) {
        setError(String(err));
        return false;
      }
    },
    [refreshRuntime, reminders, setError]
  );

  const deleteReminder = useCallback(
    async (id: number): Promise<boolean> => {
      setError('');
      try {
        const next = await deleteReminderAPI(id);
        setReminders(next);
        await refreshRuntime();
        return true;
      } catch (err) {
        setError(String(err));
        return false;
      }
    },
    [refreshRuntime, setError]
  );

  const setReminderIntervalDraft = useCallback((id: number, value: string) => {
    setReminderDrafts((prev) => ({
      ...prev,
      [id]: {
        interval: value,
        break: prev[id]?.break ?? ''
      }
    }));
  }, []);

  const setReminderBreakDraft = useCallback((id: number, value: string) => {
    setReminderDrafts((prev) => ({
      ...prev,
      [id]: {
        interval: prev[id]?.interval ?? '',
        break: value
      }
    }));
  }, []);

  const normalizeReminderIntervalDraft = useCallback(
    (id: number, raw: string, unitSec?: number) => {
      const spec = reminderFieldSpecByID(id);
      const activeUnitSec = unitSec ?? spec.intervalUnitSec;
      const { unitMin, unitMax, minSec } = deriveUnitBounds(1, undefined, activeUnitSec, activeUnitSec);
      const parsed = parseInteger(raw);
      const current = reminderByID(reminders, id);
      const fallback = clampReminderDraftValue(
        Math.round((current?.intervalSec ?? minSec) / activeUnitSec),
        unitMin,
        unitMax
      );
      const nextValue = parsed === null ? fallback : clampReminderDraftValue(parsed, unitMin, unitMax);
      setReminderIntervalDraft(id, String(nextValue));
      return nextValue;
    },
    [reminders, setReminderIntervalDraft]
  );

  const normalizeReminderBreakDraft = useCallback(
    (id: number, raw: string, unitSec?: number) => {
      const spec = reminderFieldSpecByID(id);
      const activeUnitSec = unitSec ?? spec.breakUnitSec;
      const { unitMin, unitMax, minSec } = deriveUnitBounds(1, undefined, activeUnitSec, activeUnitSec);
      const parsed = parseInteger(raw);
      const current = reminderByID(reminders, id);
      const fallback = clampReminderDraftValue(
        Math.round((current?.breakSec ?? minSec) / activeUnitSec),
        unitMin,
        unitMax
      );
      const nextValue = parsed === null ? fallback : clampReminderDraftValue(parsed, unitMin, unitMax);
      setReminderBreakDraft(id, String(nextValue));
      return nextValue;
    },
    [reminders, setReminderBreakDraft]
  );

  const commitReminderDrafts = useCallback(
    async (id: number, intervalRaw: string, breakRaw: string, intervalUnitSec?: number, breakUnitSec?: number) => {
      const spec = reminderFieldSpecByID(id);
      const current = reminderByID(reminders, id);
      if (!current) {
        setError('reminder id not found');
        return;
      }
      const activeIntervalUnitSec = intervalUnitSec ?? spec.intervalUnitSec;
      const activeBreakUnitSec = breakUnitSec ?? spec.breakUnitSec;

      const intervalBounds = deriveUnitBounds(1, undefined, activeIntervalUnitSec, activeIntervalUnitSec);
      const breakBounds = deriveUnitBounds(1, undefined, activeBreakUnitSec, activeBreakUnitSec);

      const parsedInterval = parseInteger(intervalRaw);
      if (!isReminderValueValid(parsedInterval, intervalBounds.unitMin, intervalBounds.unitMax)) {
        setError('reminder intervalSec must be > 0');
        return;
      }
      const nextInterval = parsedInterval;

      const parsedBreak = parseInteger(breakRaw);
      if (!isReminderValueValid(parsedBreak, breakBounds.unitMin, breakBounds.unitMax)) {
        setError('reminder breakSec must be > 0');
        return;
      }
      const nextBreak = parsedBreak;

      setReminderDrafts((prev) => ({
        ...prev,
        [id]: {
          interval: String(nextInterval),
          break: String(nextBreak)
        }
      }));

      const nextIntervalSec = Math.max(1, Math.round(nextInterval) * activeIntervalUnitSec);
      const nextBreakSec = Math.max(1, Math.round(nextBreak) * activeBreakUnitSec);
      const hasNoChange =
        current.intervalSec === nextIntervalSec && current.breakSec === nextBreakSec;

      if (hasNoChange) {
        return;
      }

      await applyReminderPatch(id, {
        intervalSec: nextIntervalSec,
        breakSec: nextBreakSec
      });
    },
    [applyReminderPatch, reminders]
  );

  const resetReminderDraftToStored = useCallback(
    (id: number) => {
      const current = reminderByID(reminders, id);
      if (current) {
        const intervalUnitSec = deriveCustomDraftUnitSec(current.intervalSec, 3600, 60);
        const breakUnitSec = deriveCustomDraftUnitSec(current.breakSec, 60, 1);
        setReminderDrafts((prev) => ({
          ...prev,
          [id]: {
            interval: String(Math.max(1, Math.round(current.intervalSec / intervalUnitSec))),
            break: String(Math.max(1, Math.round(current.breakSec / breakUnitSec)))
          }
        }));
        return;
      }
      setReminderDrafts((prev) => ({
        ...prev,
        [id]: {
          interval: '1',
          break: '1'
        }
      }));
    },
    [reminders]
  );

  const idleModeSelectValue = useMemo(() => {
    if (!settings) return 'off';
    if (settings.timer.mode !== 'idle_pause') return 'off';
    return String(nearestOptionValue(settings.timer.idlePauseThresholdSec, IDLE_THRESHOLD_OPTIONS));
  }, [settings]);

  const soundModeSelectValue = useMemo(() => {
    if (!settings || !settings.sound.enabled) return 'off';
    return String(nearestOptionValue(settings.sound.volume, SOUND_VOLUME_OPTIONS));
  }, [settings]);

  return {
    settings,
    reminders,
    reminderDrafts,
    launchAtLogin,
    applyLaunchAtLogin,
    applyPatch,
    applyReminderPatch,
    createReminder,
    deleteReminder,
    setReminderIntervalDraft,
    setReminderBreakDraft,
    normalizeReminderIntervalDraft,
    normalizeReminderBreakDraft,
    commitReminderDrafts,
    resetReminderDraftToStored,
    idleModeSelectValue,
    soundModeSelectValue
  };
}
