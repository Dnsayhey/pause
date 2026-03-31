import { describe, expect, it } from 'vitest';
import {
  isReminderValueValid,
  reminderFieldSpecByID,
  toDraftBreakValue,
  toDraftIntervalValue,
  toStoredBreakSec,
  toStoredIntervalSec
} from './reminderFields';

describe('reminderFields helpers', () => {
  it('returns the default spec for reminder ids', () => {
    expect(reminderFieldSpecByID(1)).toEqual({
      intervalLabelKey: 'intervalSec',
      breakLabelKey: 'breakSec',
      intervalUnitSec: 60,
      breakUnitSec: 60,
      intervalMin: 1,
      breakMin: 1
    });
  });

  it('converts stored seconds into draft values with minimum guards', () => {
    const spec = reminderFieldSpecByID(1);

    expect(toDraftIntervalValue(3600, spec)).toBe(60);
    expect(toDraftIntervalValue(0, spec)).toBe(1);
    expect(toDraftBreakValue(300, spec)).toBe(5);
    expect(toDraftBreakValue(-10, spec)).toBe(1);
  });

  it('converts draft values back into stored seconds', () => {
    const spec = reminderFieldSpecByID(1);

    expect(toStoredIntervalSec(3, spec)).toBe(180);
    expect(toStoredIntervalSec(0, spec)).toBe(1);
    expect(toStoredBreakSec(2, spec)).toBe(120);
    expect(toStoredBreakSec(-5, spec)).toBe(1);
  });

  it('validates reminder numeric bounds', () => {
    expect(isReminderValueValid(5, 1)).toBe(true);
    expect(isReminderValueValid(5, 1, 10)).toBe(true);
    expect(isReminderValueValid(0, 1)).toBe(false);
    expect(isReminderValueValid(11, 1, 10)).toBe(false);
    expect(isReminderValueValid(null, 1)).toBe(false);
  });
});
