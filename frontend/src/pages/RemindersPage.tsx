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
  onReminderIntervalDraftNormalize: (id: string, value: string) => number;
  onReminderBreakDraftChange: (id: string, value: string) => void;
  onReminderBreakDraftNormalize: (id: string, value: string) => number;
  onReminderDraftCommit: (id: string, intervalValue: string, breakValue: string) => Promise<void> | void;
  onReminderEditCancel: (id: string) => void;
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

function reminderTitleAnchorClass(reminder: ReminderConfig): string {
  if (reminder.id === 'eye') {
    return 'bg-[var(--seg-active)]';
  }
  if (reminder.id === 'stand') {
    return 'bg-[var(--text-secondary)]';
  }
  return 'bg-[var(--toggle-on)]';
}

export function RemindersPage({
  locale,
  reminders,
  reminderDrafts,
  onReminderEnabledChange,
  onReminderIntervalDraftChange,
  onReminderIntervalDraftNormalize,
  onReminderBreakDraftChange,
  onReminderBreakDraftNormalize,
  onReminderDraftCommit,
  onReminderEditCancel
}: RemindersPageProps) {
  return (
    <section className="mt-3 grid grid-cols-1 gap-3 px-2 sm:px-3 min-[721px]:grid-cols-2">
      {reminders.map((reminder) => {
        const spec = reminderFieldSpecByID(reminder.id);
        const draft = reminderDrafts[reminder.id];
        const intervalValue = draft?.interval ?? String(toDraftIntervalValue(reminder.intervalSec, spec));
        const breakValue = draft?.break ?? String(toDraftBreakValue(reminder.breakSec, spec));

        return (
          <ReminderCard
            key={reminder.id}
            locale={locale}
            title={reminderTitle(reminder, locale)}
            titleAnchorClassName={reminderTitleAnchorClass(reminder)}
            enabledLabel={t(locale, 'enabled')}
            enabled={reminder.enabled}
            onEnabledChange={(checked) => {
              onReminderEnabledChange(reminder.id, checked);
            }}
            editLabel={t(locale, 'edit')}
            doneLabel={t(locale, 'done')}
            intervalLabel={t(locale, spec.intervalLabelKey)}
            intervalValue={intervalValue}
            intervalUnitSec={spec.intervalUnitSec}
            intervalMin={spec.intervalMin}
            intervalMax={spec.intervalMax}
            onIntervalChange={(value) => {
              onReminderIntervalDraftChange(reminder.id, value);
            }}
            onIntervalNormalize={(value) => {
              onReminderIntervalDraftNormalize(reminder.id, value);
            }}
            breakLabel={t(locale, spec.breakLabelKey)}
            breakValue={breakValue}
            breakUnitSec={spec.breakUnitSec}
            breakMin={spec.breakMin}
            breakMax={spec.breakMax}
            onBreakChange={(value) => {
              onReminderBreakDraftChange(reminder.id, value);
            }}
            onBreakNormalize={(value) => {
              onReminderBreakDraftNormalize(reminder.id, value);
            }}
            onDoneEdit={() => onReminderDraftCommit(reminder.id, intervalValue, breakValue)}
            onCancelEdit={() => {
              onReminderEditCancel(reminder.id);
            }}
          />
        );
      })}
    </section>
  );
}
