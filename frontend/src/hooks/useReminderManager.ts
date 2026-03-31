import { useCallback, useEffect, useState, type Dispatch, type SetStateAction } from 'react';
import {
  createReminder as createReminderAPI,
  deleteReminder as deleteReminderAPI,
  updateReminder
} from '../api';
import { isReminderValueValid, reminderFieldSpecByID } from '../reminderFields';
import type { ReminderConfig, ReminderPatch } from '../types';
import {
  buildReminderDraft,
  clampReminderDraftValue,
  deriveUnitBounds,
  digitsOnly,
  hasNameConflict,
  isPositiveInt,
  isValidReminderType,
  normalizeReminderName,
  parseInteger,
  reminderByID,
  type ReminderDraft,
  shouldCheckNotificationCapabilityForPatch
} from './settings/helpers';

type UseReminderManagerOptions = {
  reminders: ReminderConfig[];
  setReminders: Dispatch<SetStateAction<ReminderConfig[]>>;
  setError: (message: string) => void;
  refreshRuntime: () => Promise<unknown>;
  ensureNotificationReadyForNotifyReminder: () => Promise<boolean>;
};

export function useReminderManager({
  reminders,
  setReminders,
  setError,
  refreshRuntime,
  ensureNotificationReadyForNotifyReminder
}: UseReminderManagerOptions) {
  const [reminderDrafts, setReminderDrafts] = useState<Record<number, ReminderDraft>>({});

  useEffect(() => {
    const nextDrafts: Record<number, ReminderDraft> = {};
    for (const reminder of reminders) {
      nextDrafts[reminder.id] = buildReminderDraft(reminder);
    }
    setReminderDrafts(nextDrafts);
  }, [reminders]);

  const applyReminderPatch = useCallback(
    async (id: number, patch: Omit<ReminderPatch, 'id'>) => {
      if (!isPositiveInt(id)) {
        setError('reminder id is required');
        return;
      }
      const current = reminderByID(reminders, id);
      if (!current) {
        setError('reminder id not found');
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
      if (shouldCheckNotificationCapabilityForPatch(current, nextPatch)) {
        const ready = await ensureNotificationReadyForNotifyReminder();
        if (!ready) {
          return;
        }
      }
      try {
        const next = await updateReminder({
          id,
          ...nextPatch
        });
        setReminders(next);
        await refreshRuntime();
      } catch (err) {
        setError(String(err));
      }
    },
    [ensureNotificationReadyForNotifyReminder, refreshRuntime, reminders, setError, setReminders]
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
      if (reminderType === 'notify') {
        const ready = await ensureNotificationReadyForNotifyReminder();
        if (!ready) {
          return false;
        }
      }
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
    [ensureNotificationReadyForNotifyReminder, refreshRuntime, reminders, setError, setReminders]
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
    [refreshRuntime, setError, setReminders]
  );

  const setReminderIntervalDraft = useCallback((id: number, value: string) => {
    const sanitized = digitsOnly(value);
    setReminderDrafts((prev) => ({
      ...prev,
      [id]: {
        interval: sanitized,
        break: prev[id]?.break ?? ''
      }
    }));
  }, []);

  const setReminderBreakDraft = useCallback((id: number, value: string) => {
    const sanitized = digitsOnly(value);
    setReminderDrafts((prev) => ({
      ...prev,
      [id]: {
        interval: prev[id]?.interval ?? '',
        break: sanitized
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
      const hasNoChange = current.intervalSec === nextIntervalSec && current.breakSec === nextBreakSec;

      if (hasNoChange) {
        return;
      }

      await applyReminderPatch(id, {
        intervalSec: nextIntervalSec,
        breakSec: nextBreakSec
      });
    },
    [applyReminderPatch, reminders, setError]
  );

  const resetReminderDraftToStored = useCallback(
    (id: number) => {
      const current = reminderByID(reminders, id);
      if (current) {
        setReminderDrafts((prev) => ({
          ...prev,
          [id]: buildReminderDraft(current)
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

  return {
    reminderDrafts,
    applyReminderPatch,
    createReminder,
    deleteReminder,
    setReminderIntervalDraft,
    setReminderBreakDraft,
    normalizeReminderIntervalDraft,
    normalizeReminderBreakDraft,
    commitReminderDrafts,
    resetReminderDraftToStored
  };
}
