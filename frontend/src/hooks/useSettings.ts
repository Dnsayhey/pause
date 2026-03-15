import { useCallback, useEffect, useMemo, useState } from 'react';
import { getLaunchAtLogin, getReminders, getSettings, setLaunchAtLogin, updateReminders, updateSettings } from '../api';
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

  const commitReminderIntervalDraft = useCallback(
    async (id: string, raw: string) => {
      const spec = reminderFieldSpecByID(id);
      const parsed = parseInteger(raw);
      const current = reminderByID(reminders, id);
      const fallback = toDraftIntervalValue(current?.intervalSec ?? spec.intervalMin * spec.intervalUnitSec, spec);
      const nextValue = isReminderValueValid(parsed, spec.intervalMin, spec.intervalMax) ? parsed : fallback;

      setReminderIntervalDraft(id, String(nextValue));
      await applyReminderPatch(id, { intervalSec: toStoredIntervalSec(nextValue, spec) });
    },
    [applyReminderPatch, reminders, setReminderIntervalDraft]
  );

  const commitReminderBreakDraft = useCallback(
    async (id: string, raw: string) => {
      const spec = reminderFieldSpecByID(id);
      const parsed = parseInteger(raw);
      const current = reminderByID(reminders, id);
      const fallback = toDraftBreakValue(current?.breakSec ?? spec.breakMin * spec.breakUnitSec, spec);
      const nextValue = isReminderValueValid(parsed, spec.breakMin, spec.breakMax) ? parsed : fallback;

      setReminderBreakDraft(id, String(nextValue));
      await applyReminderPatch(id, { breakSec: toStoredBreakSec(nextValue, spec) });
    },
    [applyReminderPatch, reminders, setReminderBreakDraft]
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
    setReminderIntervalDraft,
    setReminderBreakDraft,
    commitReminderIntervalDraft,
    commitReminderBreakDraft,
    idleModeSelectValue,
    soundModeSelectValue
  };
}
