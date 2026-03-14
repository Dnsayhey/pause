import { useCallback, useEffect, useRef, useState } from 'react';
import { skipCurrentBreak } from './api';
import { resolveLocale, t } from './i18n';
import { HeroHeader } from './components/HeroHeader';
import { ReminderCard } from './components/ReminderCard';
import { SystemSettingsCard } from './components/SystemSettingsCard';
import { BreakOverlay } from './components/BreakOverlay';
import { InlineError } from './components/ui';
import { useRuntimePolling } from './hooks/useRuntimePolling';
import { useSettings } from './hooks/useSettings';

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

export function App() {
  const [error, setError] = useState('');
  const titleRef = useRef<HTMLHeadingElement | null>(null);
  const hasAssignedInitialFocusRef = useRef(false);

  const { runtime, setRuntime, refreshRuntime } = useRuntimePolling({
    setError
  });

  const {
    settings,
    launchAtLogin,
    applyLaunchAtLogin,
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
  } = useSettings({
    setError,
    refreshRuntime
  });

  useEffect(() => {
    const platformClass = detectPlatformClass();
    document.body.dataset.platform = platformClass;
    return () => {
      delete document.body.dataset.platform;
    };
  }, []);

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
          {t(resolveLocale('auto'), 'loading')}
          {error && <InlineError message={error} />}
          </div>
        </div>
      </div>
    );
  }

  const locale = resolveLocale(settings.ui.language);
  const overlayActive = Boolean(
    !runtime.overlayNative &&
      runtime.currentSession &&
      runtime.currentSession.status === 'resting'
  );

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
          <ReminderCard
            title={t(locale, 'eyeReminder')}
            enabledLabel={t(locale, 'enabled')}
            enabled={settings.eye.enabled}
            onEnabledChange={(checked) => {
              void applyPatch({ eye: { enabled: checked } });
            }}
            intervalLabel={t(locale, 'eyeIntervalMin')}
            intervalValue={eyeIntervalMinDraft}
            intervalMin={1}
            onIntervalChange={setEyeIntervalMinDraft}
            onIntervalCommit={commitEyeIntervalDraft}
            breakLabel={t(locale, 'eyeBreakSec')}
            breakValue={eyeBreakSecDraft}
            breakMin={10}
            breakMax={60}
            onBreakChange={setEyeBreakSecDraft}
            onBreakCommit={commitEyeBreakDraft}
          />

          <ReminderCard
            title={t(locale, 'standReminder')}
            enabledLabel={t(locale, 'enabled')}
            enabled={settings.stand.enabled}
            onEnabledChange={(checked) => {
              void applyPatch({ stand: { enabled: checked } });
            }}
            intervalLabel={t(locale, 'standIntervalHour')}
            intervalValue={standIntervalHourDraft}
            intervalMin={1}
            onIntervalChange={setStandIntervalHourDraft}
            onIntervalCommit={commitStandIntervalDraft}
            breakLabel={t(locale, 'standBreakMin')}
            breakValue={standBreakMinDraft}
            breakMin={1}
            breakMax={10}
            onBreakChange={setStandBreakMinDraft}
            onBreakCommit={commitStandBreakDraft}
          />
        </section>

        <SystemSettingsCard
          locale={locale}
          settings={settings}
          launchAtLogin={launchAtLogin}
          idleModeSelectValue={idleModeSelectValue}
          soundModeSelectValue={soundModeSelectValue}
          onLaunchAtLoginChange={applyLaunchAtLogin}
          onPatch={applyPatch}
        />

        {overlayActive && (
          <BreakOverlay
            locale={locale}
            runtime={runtime}
            onSkip={() => {
              setError('');
              void skipCurrentBreak()
                .then(setRuntime)
                .catch((e) => setError(String(e)));
            }}
          />
        )}
        </div>
      </div>
    </div>
  );
}
