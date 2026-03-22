import { useCallback, useEffect, useMemo, useState } from 'react';
import { createReminder as createReminderAPI, getLaunchAtLogin, getReminders, getSettings, setLaunchAtLogin, updateReminders, updateSettings } from '../api';
import {
  isReminderValueValid,
  reminderFieldSpecByID,
  toDraftBreakValue,
  toDraftIntervalValue,
  toStoredBreakSec,
  toStoredIntervalSec
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

function reminderByID(reminders: ReminderConfig[], id: string) {
  return reminders.find((reminder) => reminder.id === id);
}

export function useSettings({ setError, refreshRuntime }: UseSettingsOptions) {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [reminders, setReminders] = useState<ReminderConfig[]>([]);
  const [reminderDrafts, setReminderDrafts] = useState<Record<string, ReminderDraft>>({});
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
    const nextDrafts: Record<string, ReminderDraft> = {};
    for (const reminder of reminders) {
      const spec = reminderFieldSpecByID(reminder.id);
      nextDrafts[reminder.id] = {
        interval: String(toDraftIntervalValue(reminder.intervalSec, spec)),
        break: String(toDraftBreakValue(reminder.breakSec, spec))
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
    async (id: string, patch: Omit<ReminderPatch, 'id'>) => {
      setError('');
      try {
        const next = await updateReminders([
          {
            id,
            ...patch
          }
        ]);
        setReminders(next);
        await refreshRuntime();
      } catch (err) {
        setError(String(err));
      }
    },
    [refreshRuntime, setError]
  );

  const createReminder = useCallback(
    async (name: string, intervalSec: number, breakSec: number, reminderType: 'rest' | 'notify'): Promise<boolean> => {
      setError('');
      try {
        const next = await createReminderAPI({
          name,
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
    [refreshRuntime, setError]
  );

  const setReminderIntervalDraft = useCallback((id: string, value: string) => {
    setReminderDrafts((prev) => ({
      ...prev,
      [id]: {
        interval: value,
        break: prev[id]?.break ?? ''
      }
    }));
  }, []);

  const setReminderBreakDraft = useCallback((id: string, value: string) => {
    setReminderDrafts((prev) => ({
      ...prev,
      [id]: {
        interval: prev[id]?.interval ?? '',
        break: value
      }
    }));
  }, []);

  const normalizeReminderIntervalDraft = useCallback(
    (id: string, raw: string) => {
      const spec = reminderFieldSpecByID(id);
      const parsed = parseInteger(raw);
      const current = reminderByID(reminders, id);
      const fallback = toDraftIntervalValue(current?.intervalSec ?? spec.intervalMin * spec.intervalUnitSec, spec);
      const nextValue = parsed === null ? fallback : clampReminderDraftValue(parsed, spec.intervalMin, spec.intervalMax);
      setReminderIntervalDraft(id, String(nextValue));
      return nextValue;
    },
    [reminders, setReminderIntervalDraft]
  );

  const normalizeReminderBreakDraft = useCallback(
    (id: string, raw: string) => {
      const spec = reminderFieldSpecByID(id);
      const parsed = parseInteger(raw);
      const current = reminderByID(reminders, id);
      const fallback = toDraftBreakValue(current?.breakSec ?? spec.breakMin * spec.breakUnitSec, spec);
      const nextValue = parsed === null ? fallback : clampReminderDraftValue(parsed, spec.breakMin, spec.breakMax);
      setReminderBreakDraft(id, String(nextValue));
      return nextValue;
    },
    [reminders, setReminderBreakDraft]
  );

  const commitReminderDrafts = useCallback(
    async (id: string, intervalRaw: string, breakRaw: string) => {
      const spec = reminderFieldSpecByID(id);
      const current = reminderByID(reminders, id);

      const parsedInterval = parseInteger(intervalRaw);
      const fallbackInterval = toDraftIntervalValue(current?.intervalSec ?? spec.intervalMin * spec.intervalUnitSec, spec);
      const nextInterval =
        parsedInterval === null
          ? fallbackInterval
          : isReminderValueValid(parsedInterval, spec.intervalMin, spec.intervalMax)
            ? parsedInterval
            : clampReminderDraftValue(parsedInterval, spec.intervalMin, spec.intervalMax);

      const parsedBreak = parseInteger(breakRaw);
      const fallbackBreak = toDraftBreakValue(current?.breakSec ?? spec.breakMin * spec.breakUnitSec, spec);
      const nextBreak =
        parsedBreak === null
          ? fallbackBreak
          : isReminderValueValid(parsedBreak, spec.breakMin, spec.breakMax)
            ? parsedBreak
            : clampReminderDraftValue(parsedBreak, spec.breakMin, spec.breakMax);

      setReminderDrafts((prev) => ({
        ...prev,
        [id]: {
          interval: String(nextInterval),
          break: String(nextBreak)
        }
      }));

      const nextIntervalSec = toStoredIntervalSec(nextInterval, spec);
      const nextBreakSec = toStoredBreakSec(nextBreak, spec);
      const hasNoChange =
        current !== undefined && current.intervalSec === nextIntervalSec && current.breakSec === nextBreakSec;

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
    (id: string) => {
      const spec = reminderFieldSpecByID(id);
      const current = reminderByID(reminders, id);
      const interval = toDraftIntervalValue(current?.intervalSec ?? spec.intervalMin * spec.intervalUnitSec, spec);
      const breakValue = toDraftBreakValue(current?.breakSec ?? spec.breakMin * spec.breakUnitSec, spec);
      setReminderDrafts((prev) => ({
        ...prev,
        [id]: {
          interval: String(interval),
          break: String(breakValue)
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
