import { localizeReason, t, type Locale } from '../i18n';
import { reminderFieldSpecByID, toDraftBreakValue, toDraftIntervalValue } from '../reminderFields';
import type { ReminderConfig } from '../types';
import { ReminderCard } from '../components/ReminderCard';

type ReminderDrafts = Record<string, { interval: string; break: string }>;

type RemindersPageProps = {
  locale: Locale;
  reminders: ReminderConfig[];
  reminderDrafts: ReminderDrafts;
  onReminderEnabledChange: (id: string, enabled: boolean) => void;
  onReminderIntervalDraftChange: (id: string, value: string) => void;
  onReminderIntervalCommit: (id: string, value: string) => Promise<void>;
  onReminderBreakDraftChange: (id: string, value: string) => void;
  onReminderBreakCommit: (id: string, value: string) => Promise<void>;
};

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
  reminderDrafts,
  onReminderEnabledChange,
  onReminderIntervalDraftChange,
  onReminderIntervalCommit,
  onReminderBreakDraftChange,
  onReminderBreakCommit
}: RemindersPageProps) {
  return (
    <section className="mt-3 grid grid-cols-1 gap-3 min-[721px]:grid-cols-2">
      {reminders.map((reminder) => {
        const spec = reminderFieldSpecByID(reminder.id);
        const draft = reminderDrafts[reminder.id];
        const intervalValue = draft?.interval ?? String(toDraftIntervalValue(reminder.intervalSec, spec));
        const breakValue = draft?.break ?? String(toDraftBreakValue(reminder.breakSec, spec));

        return (
          <ReminderCard
            key={reminder.id}
            title={reminderTitle(reminder, locale)}
            enabledLabel={t(locale, 'enabled')}
            enabled={reminder.enabled}
            onEnabledChange={(checked) => {
              onReminderEnabledChange(reminder.id, checked);
            }}
            intervalLabel={t(locale, spec.intervalLabelKey)}
            intervalValue={intervalValue}
            intervalMin={spec.intervalMin}
            intervalMax={spec.intervalMax}
            onIntervalChange={(value) => {
              onReminderIntervalDraftChange(reminder.id, value);
            }}
            onIntervalCommit={(value) => onReminderIntervalCommit(reminder.id, value)}
            breakLabel={t(locale, spec.breakLabelKey)}
            breakValue={breakValue}
            breakMin={spec.breakMin}
            breakMax={spec.breakMax}
            onBreakChange={(value) => {
              onReminderBreakDraftChange(reminder.id, value);
            }}
            onBreakCommit={(value) => onReminderBreakCommit(reminder.id, value)}
          />
        );
      })}
    </section>
  );
}
