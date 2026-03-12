import { useCallback, useEffect, useMemo, useState } from 'react';
import { getSettings, updateSettings } from '../api';
import type { Settings, SettingsPatch } from '../types';

const EYE_DEFAULT_INTERVAL_MIN = 20;
const EYE_DEFAULT_BREAK_SEC = 20;
const STAND_DEFAULT_INTERVAL_HOUR = 1;
const STAND_DEFAULT_BREAK_MIN = 5;
const IDLE_THRESHOLD_OPTIONS = [60, 300, 600, 1800, 3600, 7200] as const;
const SOUND_VOLUME_OPTIONS = [20, 40, 60, 80, 100] as const;

type UseSettingsOptions = {
  setError: (message: string) => void;
  refreshRuntime: () => Promise<unknown>;
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

export function useSettings({ setError, refreshRuntime }: UseSettingsOptions) {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [eyeIntervalMinDraft, setEyeIntervalMinDraft] = useState(String(EYE_DEFAULT_INTERVAL_MIN));
  const [eyeBreakSecDraft, setEyeBreakSecDraft] = useState(String(EYE_DEFAULT_BREAK_SEC));
  const [standIntervalHourDraft, setStandIntervalHourDraft] = useState(String(STAND_DEFAULT_INTERVAL_HOUR));
  const [standBreakMinDraft, setStandBreakMinDraft] = useState(String(STAND_DEFAULT_BREAK_MIN));

  useEffect(() => {
    let mounted = true;

    const loadSettings = async () => {
      try {
        const next = await getSettings();
        if (mounted) {
          setSettings(next);
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

  useEffect(() => {
    if (!settings) return;
    setEyeIntervalMinDraft(String(Math.max(1, Math.round(settings.eye.intervalSec / 60))));
    setEyeBreakSecDraft(String(settings.eye.breakSec));
    setStandIntervalHourDraft(String(Math.max(1, Math.round(settings.stand.intervalSec / 3600))));
    setStandBreakMinDraft(String(Math.max(1, Math.round(settings.stand.breakSec / 60))));
  }, [settings]);

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

  const commitEyeIntervalDraft = useCallback(
    async (raw: string) => {
      const intervalMin = parseInteger(raw);
      const valid = intervalMin !== null && intervalMin >= 1;
      if (!valid) {
        setEyeIntervalMinDraft(String(EYE_DEFAULT_INTERVAL_MIN));
        await applyPatch({
          eye: {
            intervalSec: EYE_DEFAULT_INTERVAL_MIN * 60
          }
        });
        return;
      }
      setEyeIntervalMinDraft(String(intervalMin));
      await applyPatch({
        eye: {
          intervalSec: intervalMin * 60
        }
      });
    },
    [applyPatch]
  );

  const commitEyeBreakDraft = useCallback(
    async (raw: string) => {
      const breakSec = parseInteger(raw);
      const valid = breakSec !== null && breakSec >= 10 && breakSec <= 60;
      if (!valid) {
        setEyeBreakSecDraft(String(EYE_DEFAULT_BREAK_SEC));
        await applyPatch({
          eye: {
            breakSec: EYE_DEFAULT_BREAK_SEC
          }
        });
        return;
      }
      setEyeBreakSecDraft(String(breakSec));
      await applyPatch({
        eye: {
          breakSec
        }
      });
    },
    [applyPatch]
  );

  const commitStandIntervalDraft = useCallback(
    async (raw: string) => {
      const intervalHour = parseInteger(raw);
      const valid = intervalHour !== null && intervalHour >= 1;
      if (!valid) {
        setStandIntervalHourDraft(String(STAND_DEFAULT_INTERVAL_HOUR));
        await applyPatch({
          stand: {
            intervalSec: STAND_DEFAULT_INTERVAL_HOUR * 3600
          }
        });
        return;
      }
      setStandIntervalHourDraft(String(intervalHour));
      await applyPatch({
        stand: {
          intervalSec: intervalHour * 3600
        }
      });
    },
    [applyPatch]
  );

  const commitStandBreakDraft = useCallback(
    async (raw: string) => {
      const breakMin = parseInteger(raw);
      const valid = breakMin !== null && breakMin >= 1 && breakMin <= 10;
      if (!valid) {
        setStandBreakMinDraft(String(STAND_DEFAULT_BREAK_MIN));
        await applyPatch({
          stand: {
            breakSec: STAND_DEFAULT_BREAK_MIN * 60
          }
        });
        return;
      }
      setStandBreakMinDraft(String(breakMin));
      await applyPatch({
        stand: {
          breakSec: breakMin * 60
        }
      });
    },
    [applyPatch]
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
    applyPatch,
    eyeIntervalMinDraft,
    setEyeIntervalMinDraft,
    eyeBreakSecDraft,
    setEyeBreakSecDraft,
    standIntervalHourDraft,
    setStandIntervalHourDraft,
    standBreakMinDraft,
    setStandBreakMinDraft,
    commitEyeIntervalDraft,
    commitEyeBreakDraft,
    commitStandIntervalDraft,
    commitStandBreakDraft,
    idleModeSelectValue,
    soundModeSelectValue
  };
}
