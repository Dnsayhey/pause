import type { TranslationKey } from './i18n';

export type ReminderFieldSpec = {
  titleKey?: TranslationKey;
  intervalLabelKey: TranslationKey;
  breakLabelKey: TranslationKey;
  intervalUnitSec: number;
  breakUnitSec: number;
  intervalMin: number;
  intervalMax?: number;
  breakMin: number;
  breakMax?: number;
};

const DEFAULT_SPEC: ReminderFieldSpec = {
  intervalLabelKey: 'intervalSec',
  breakLabelKey: 'breakSec',
  intervalUnitSec: 1,
  breakUnitSec: 1,
  intervalMin: 1,
  breakMin: 1
};

export function reminderFieldSpecByID(id: string): ReminderFieldSpec {
  if (id === 'eye') {
    return {
      titleKey: 'eyeReminder',
      intervalLabelKey: 'eyeIntervalMin',
      breakLabelKey: 'eyeBreakSec',
      intervalUnitSec: 60,
      breakUnitSec: 1,
      intervalMin: 1,
      breakMin: 10,
      breakMax: 60
    };
  }
  if (id === 'stand') {
    return {
      titleKey: 'standReminder',
      intervalLabelKey: 'standIntervalHour',
      breakLabelKey: 'standBreakMin',
      intervalUnitSec: 3600,
      breakUnitSec: 60,
      intervalMin: 1,
      breakMin: 1,
      breakMax: 10
    };
  }
  return DEFAULT_SPEC;
}

export function toDraftIntervalValue(intervalSec: number, spec: ReminderFieldSpec): number {
  const normalized = Math.round(Math.max(1, intervalSec) / spec.intervalUnitSec);
  return Math.max(spec.intervalMin, normalized);
}

export function toDraftBreakValue(breakSec: number, spec: ReminderFieldSpec): number {
  const normalized = Math.round(Math.max(1, breakSec) / spec.breakUnitSec);
  return Math.max(spec.breakMin, normalized);
}

export function toStoredIntervalSec(intervalValue: number, spec: ReminderFieldSpec): number {
  return Math.max(1, Math.round(intervalValue) * spec.intervalUnitSec);
}

export function toStoredBreakSec(breakValue: number, spec: ReminderFieldSpec): number {
  return Math.max(1, Math.round(breakValue) * spec.breakUnitSec);
}

export function isReminderValueValid(value: number | null, min: number, max?: number): value is number {
  if (value === null || Number.isNaN(value)) {
    return false;
  }
  if (value < min) {
    return false;
  }
  if (max !== undefined && value > max) {
    return false;
  }
  return true;
}
