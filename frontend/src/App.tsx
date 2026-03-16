import { useCallback, useEffect, useRef, useState } from 'react';
import { localizeReason, resolveLocale, t } from './i18n';
import { HeroHeader } from './components/HeroHeader';
import { ReminderCard } from './components/ReminderCard';
import { SystemSettingsCard } from './components/SystemSettingsCard';
import { InlineError } from './components/ui';
import { useRuntimePolling } from './hooks/useRuntimePolling';
import { useSettings } from './hooks/useSettings';
import { reminderFieldSpecByID, toDraftBreakValue, toDraftIntervalValue } from './reminderFields';
import type { ReminderConfig } from './types';

function detectPlatformClass(): string {
  if (typeof navigator === 'undefined') return 'other';
  const nav = navigator as Navigator & { userAgentData?: { platform?: string } };
  const platform = (nav.userAgentData?.platform || navigator.platform || '').toLowerCase();
  const ua = (navigator.userAgent || '').toLowerCase();
  if (platform.includes('mac') || ua.includes('mac os')) return 'mac';
  if (platform.includes('win') || ua.includes('windows')) return 'win';
  if (platform.includes('linux') || ua.includes('linux')) return 'linux';
  return 'other';
}

function reminderTitle(reminder: ReminderConfig, locale: ReturnType<typeof resolveLocale>): string {
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

export function App() {
  const platformClass = detectPlatformClass();
  const [error, setError] = useState('');
  const titleRef = useRef<HTMLHeadingElement | null>(null);
  const hasAssignedInitialFocusRef = useRef(false);

  const { runtime, refreshRuntime } = useRuntimePolling({
    setError
  });

  const {
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
  } = useSettings({
    setError,
    refreshRuntime
  });

  useEffect(() => {
    document.body.dataset.platform = platformClass;
    return () => {
      delete document.body.dataset.platform;
    };
  }, [platformClass]);

  useEffect(() => {
    const language = runtime?.effectiveLanguage;
    const theme = runtime?.effectiveTheme;
    if (language === 'zh-CN' || language === 'en-US') {
      document.body.dataset.language = language;
    } else {
      delete document.body.dataset.language;
    }
    if (theme === 'light' || theme === 'dark') {
      document.body.dataset.theme = theme;
    } else {
      delete document.body.dataset.theme;
    }
    return () => {
      delete document.body.dataset.language;
      delete document.body.dataset.theme;
    };
  }, [runtime?.effectiveLanguage, runtime?.effectiveTheme]);

  const focusTitleIfNeeded = useCallback(() => {
    if (hasAssignedInitialFocusRef.current) return;
    if (document.visibilityState !== 'visible') return;
    if (!settings || !runtime) return;
    if (!titleRef.current) return;
    hasAssignedInitialFocusRef.current = true;
    titleRef.current.focus({ preventScroll: true });
  }, [settings, runtime]);

  useEffect(() => {
    const handleVisibilityChange = () => {
      window.requestAnimationFrame(() => {
        focusTitleIfNeeded();
      });
    };

    handleVisibilityChange();
    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [focusTitleIfNeeded]);

  if (!settings || !runtime) {
    return (
      <div className="h-full select-none overflow-hidden">
      <div className="h-7 select-none [--wails-draggable:drag]" />
      <div className="h-[calc(100%-1.75rem)] overflow-hidden">
          <div className="mx-auto max-w-[840px] p-[12px] sm:px-5 sm:py-[10px]">
          {t(resolveLocale(undefined), 'loading')}
          {error && <InlineError message={error} />}
          </div>
        </div>
      </div>
    );
  }

  const locale = resolveLocale(runtime.effectiveLanguage);

  return (
    <div className="h-full select-none overflow-hidden">
      <div className="h-7 select-none [--wails-draggable:drag]" />
      <div className="h-[calc(100%-1.75rem)] overflow-hidden">
        <div className="mx-auto max-w-[840px] p-[12px] sm:px-5 sm:py-[10px]">
        <HeroHeader
          locale={locale}
          language={settings.ui.language}
          titleRef={titleRef}
          onLanguageChange={(language) => {
            void applyPatch({ ui: { language } });
          }}
        />

        {error && <InlineError message={error} />}

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
                  void applyReminderPatch(reminder.id, { enabled: checked });
                }}
                intervalLabel={t(locale, spec.intervalLabelKey)}
                intervalValue={intervalValue}
                intervalMin={spec.intervalMin}
                intervalMax={spec.intervalMax}
                onIntervalChange={(value) => {
                  setReminderIntervalDraft(reminder.id, value);
                }}
                onIntervalCommit={(value) => commitReminderIntervalDraft(reminder.id, value)}
                breakLabel={t(locale, spec.breakLabelKey)}
                breakValue={breakValue}
                breakMin={spec.breakMin}
                breakMax={spec.breakMax}
                onBreakChange={(value) => {
                  setReminderBreakDraft(reminder.id, value);
                }}
                onBreakCommit={(value) => commitReminderBreakDraft(reminder.id, value)}
              />
            );
          })}
        </section>

        <SystemSettingsCard
          locale={locale}
          settings={settings}
          launchAtLogin={launchAtLogin}
          idleModeSelectValue={idleModeSelectValue}
          soundModeSelectValue={soundModeSelectValue}
          showTrayCountdownOption={platformClass !== 'win'}
          onLaunchAtLoginChange={applyLaunchAtLogin}
          onPatch={applyPatch}
        />
        </div>
      </div>
    </div>
  );
}
