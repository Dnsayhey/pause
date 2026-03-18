import { Suspense, lazy, useCallback, useEffect, useRef, useState } from 'react';
import { Navigate, NavLink, Route, Routes } from 'react-router-dom';
import { resolveLocale, t } from './i18n';
import { HeroHeader } from './components/HeroHeader';
import { InlineError } from './components/ui';
import { useRuntimePolling } from './hooks/useRuntimePolling';
import { useSettings } from './hooks/useSettings';
import { RemindersPage } from './pages/RemindersPage';
import { SettingsPage } from './pages/SettingsPage';

const AnalyticsPage = lazy(async () => import('./pages/AnalyticsPage').then((module) => ({ default: module.AnalyticsPage })));

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
        <div className="h-[calc(100%-1.75rem)] overflow-y-auto">
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
      <div className="app-scroll-area h-[calc(100%-1.75rem)] overflow-y-auto">
        <div className="mx-auto max-w-[840px] p-[12px] sm:px-5 sm:py-[10px]">
          <HeroHeader
            locale={locale}
            titleRef={titleRef}
            actions={
              <div className="inline-flex rounded-full border border-[var(--glass-control-border)] bg-[var(--glass-control-bg)] p-1 [backdrop-filter:blur(var(--surface-blur))_saturate(var(--surface-sat))]">
                <NavLink
                  to="/reminders"
                  className={({ isActive }) =>
                    `rounded-full px-3 py-1 text-xs no-underline transition-colors ${
                      isActive ? 'bg-[#0f826b] text-white' : 'text-[var(--control-text)] hover:bg-[var(--control-hover-bg)]'
                    }`
                  }
                >
                  {t(locale, 'navReminders')}
                </NavLink>
                <NavLink
                  to="/analytics"
                  className={({ isActive }) =>
                    `rounded-full px-3 py-1 text-xs no-underline transition-colors ${
                      isActive ? 'bg-[#0f826b] text-white' : 'text-[var(--control-text)] hover:bg-[var(--control-hover-bg)]'
                    }`
                  }
                >
                  {t(locale, 'navAnalytics')}
                </NavLink>
                <NavLink
                  to="/settings"
                  className={({ isActive }) =>
                    `rounded-full px-3 py-1 text-xs no-underline transition-colors ${
                      isActive ? 'bg-[#0f826b] text-white' : 'text-[var(--control-text)] hover:bg-[var(--control-hover-bg)]'
                    }`
                  }
                >
                  {t(locale, 'navSettings')}
                </NavLink>
              </div>
            }
          />

          {error && <InlineError message={error} />}

          <Routes>
            <Route
              path="/reminders"
              element={
                <RemindersPage
                  locale={locale}
                  reminders={reminders}
                  reminderDrafts={reminderDrafts}
                  onReminderEnabledChange={(id, enabled) => {
                    void applyReminderPatch(id, { enabled });
                  }}
                  onReminderIntervalDraftChange={setReminderIntervalDraft}
                  onReminderIntervalCommit={commitReminderIntervalDraft}
                  onReminderBreakDraftChange={setReminderBreakDraft}
                  onReminderBreakCommit={commitReminderBreakDraft}
                />
              }
            />
            <Route
              path="/analytics"
              element={
                <Suspense fallback={<p className="mt-3 text-sm text-[var(--text-secondary)]">{t(locale, 'analyticsLoading')}</p>}>
                  <AnalyticsPage locale={locale} />
                </Suspense>
              }
            />
            <Route
              path="/settings"
              element={
                <SettingsPage
                  locale={locale}
                  settings={settings}
                  launchAtLogin={launchAtLogin}
                  idleModeSelectValue={idleModeSelectValue}
                  soundModeSelectValue={soundModeSelectValue}
                  showTrayCountdownOption={platformClass !== 'win'}
                  onLaunchAtLoginChange={applyLaunchAtLogin}
                  onPatch={applyPatch}
                />
              }
            />
            <Route path="*" element={<Navigate to="/reminders" replace />} />
          </Routes>
        </div>
      </div>
    </div>
  );
}
