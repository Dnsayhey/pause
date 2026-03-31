import type { NotificationCapability, NotificationProductState, ReminderConfig, ReminderPatch } from '../../types';

export const NOTIFICATION_ERROR_PERMISSION_DENIED = 'ERR_NOTIFICATION_PERMISSION_DENIED';
export const NOTIFICATION_ERROR_PERMISSION_REQUIRED = 'ERR_NOTIFICATION_PERMISSION_REQUIRED';
export const NOTIFICATION_ERROR_UNAVAILABLE = 'ERR_NOTIFICATION_UNAVAILABLE';

export type ReminderDraft = {
  interval: string;
  break: string;
};

export function digitsOnly(text: string): string {
  return text.replace(/\D+/g, '');
}

export function parseInteger(text: string): number | null {
  const trimmed = text.trim();
  if (trimmed === '') return null;
  if (!/^[0-9]+$/.test(trimmed)) return null;
  const value = Number(trimmed);
  if (!Number.isSafeInteger(value)) return null;
  return value;
}

export function deriveUnitBounds(min: number, max: number | undefined, baseUnitSec: number, activeUnitSec: number) {
  const minSec = min * baseUnitSec;
  const maxSec = max === undefined ? undefined : max * baseUnitSec;
  const unitMin = Math.max(1, Math.ceil(minSec / activeUnitSec));
  const unitMax =
    maxSec === undefined ? undefined : Math.max(unitMin, Math.floor(maxSec / activeUnitSec));
  return { unitMin, unitMax, minSec };
}

export function deriveCustomDraftUnitSec(totalSec: number, primaryUnitSec: number, secondaryUnitSec: number): number {
  if (totalSec >= primaryUnitSec && totalSec % primaryUnitSec === 0) {
    return primaryUnitSec;
  }
  return secondaryUnitSec;
}

export function buildReminderDraft(reminder: ReminderConfig): ReminderDraft {
  const intervalUnitSec = deriveCustomDraftUnitSec(reminder.intervalSec, 3600, 60);
  const breakUnitSec = deriveCustomDraftUnitSec(reminder.breakSec, 60, 1);
  return {
    interval: String(Math.max(1, Math.round(reminder.intervalSec / intervalUnitSec))),
    break: String(Math.max(1, Math.round(reminder.breakSec / breakUnitSec)))
  };
}

export function clampReminderDraftValue(value: number, min: number, max?: number): number {
  if (value < min) return min;
  if (max !== undefined && value > max) return max;
  return value;
}

export function nearestOptionValue(value: number, options: readonly number[]): number {
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

export function reminderByID(reminders: ReminderConfig[], id: number) {
  return reminders.find((reminder) => reminder.id === id);
}

export function normalizeReminderName(name: string): string {
  return name.trim();
}

export function isValidReminderType(value: unknown): value is 'rest' | 'notify' {
  return value === 'rest' || value === 'notify';
}

export function isPositiveInt(value: unknown): value is number {
  return Number.isInteger(value) && Number(value) > 0;
}

export function hasNameConflict(reminders: ReminderConfig[], name: string, excludeID?: number): boolean {
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

export function notificationProductStateFromCapability(
  capability: NotificationCapability | null
): NotificationProductState | null {
  if (!capability) {
    return null;
  }
  if (capability.permissionState === 'authorized') {
    return 'available';
  }
  if (capability.permissionState === 'not_determined') {
    return 'pending';
  }
  return 'unavailable';
}

export function notificationErrorCodeFromCapability(capability: NotificationCapability | null): string {
  const productState = notificationProductStateFromCapability(capability);
  if (productState === 'pending') {
    return NOTIFICATION_ERROR_PERMISSION_REQUIRED;
  }
  if (productState !== 'unavailable') {
    return '';
  }
  if (capability?.permissionState === 'denied') {
    return NOTIFICATION_ERROR_PERMISSION_DENIED;
  }
  return NOTIFICATION_ERROR_UNAVAILABLE;
}

export function shouldCheckNotificationCapabilityForPatch(
  current: ReminderConfig,
  patch: Omit<ReminderPatch, 'id'>
): boolean {
  const nextEnabled = patch.enabled ?? current.enabled;
  const nextReminderType = patch.reminderType ?? current.reminderType;
  if (!nextEnabled || nextReminderType !== 'notify') {
    return false;
  }
  return current.reminderType !== 'notify' || !current.enabled;
}
