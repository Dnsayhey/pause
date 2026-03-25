import { Suspense, lazy, useCallback, useEffect, useRef, useState } from 'react';
import { Navigate, NavLink, Route, Routes, useLocation } from 'react-router-dom';
import { closeWindow } from './api';
import { resolveLocale, t } from './i18n';
import { HeroHeader } from './components/HeroHeader';
import { CustomScrollArea, InlineError } from './components/ui';
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
  const location = useLocation();
  const platformClass = detectPlatformClass();
  const isWindows = platformClass === 'win';
  const dragBarHeightClass = isWindows ? 'h-8' : 'h-7';
  const contentHeightClass = isWindows ? 'h-[calc(100%-2rem)]' : 'h-[calc(100%-1.75rem)]';
  const windowsCloseButtonBaseClass =
    'absolute right-0 top-0 inline-flex h-full w-[46px] items-center justify-center rounded-none border-0 transition-colors duration-120 ease-out focus-visible:outline-none [--wails-draggable:no-drag]';
  const fallbackLocale = resolveLocale(undefined);
  const [error, setError] = useState('');
  const [isWindowsCloseHovered, setIsWindowsCloseHovered] = useState(false);
  const [isWindowsClosePressed, setIsWindowsClosePressed] = useState(false);
  const [createPanelRequestId, setCreatePanelRequestId] = useState(0);
  const [createPanelAnchor, setCreatePanelAnchor] = useState<{ top: number; right: number } | null>(null);
  const addReminderButtonRef = useRef<HTMLButtonElement | null>(null);
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
    createReminder,
    deleteReminder,
    setReminderIntervalDraft,
    setReminderBreakDraft,
    normalizeReminderIntervalDraft,
    normalizeReminderBreakDraft,
    commitReminderDrafts,
    resetReminderDraftToStored,
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

  const resetWindowsCloseVisualState = useCallback(() => {
    setIsWindowsCloseHovered(false);
    setIsWindowsClosePressed(false);
  }, []);

  useEffect(() => {
    if (!isWindows) return;
    const handleVisibilityChange = () => {
      resetWindowsCloseVisualState();
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    window.addEventListener('blur', resetWindowsCloseVisualState);
    window.addEventListener('focus', resetWindowsCloseVisualState);
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      window.removeEventListener('blur', resetWindowsCloseVisualState);
      window.removeEventListener('focus', resetWindowsCloseVisualState);
    };
  }, [isWindows, resetWindowsCloseVisualState]);

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

  const windowsCloseButtonStateClass = isWindowsClosePressed
    ? 'bg-[#c50f1f] text-white'
    : isWindowsCloseHovered
      ? 'bg-[#e81123] text-white'
      : 'bg-transparent text-[var(--text-secondary)]';
  const windowsCloseButtonClass = `${windowsCloseButtonBaseClass} ${windowsCloseButtonStateClass}`;

  if (!settings || !runtime) {
    return (
      <div className="h-full select-none overflow-hidden">
        <div className={`${dragBarHeightClass} relative select-none [--wails-draggable:drag]`}>
          {isWindows && (
            <button
              type="button"
              className={windowsCloseButtonClass}
              aria-label={t(fallbackLocale, 'close')}
              title={t(fallbackLocale, 'close')}
              onPointerEnter={() => {
                setIsWindowsCloseHovered(true);
              }}
              onPointerLeave={() => {
                resetWindowsCloseVisualState();
              }}
              onPointerDown={(event) => {
                if (event.button !== 0) return;
                setIsWindowsClosePressed(true);
              }}
              onPointerUp={() => {
                setIsWindowsClosePressed(false);
              }}
              onPointerCancel={() => {
                resetWindowsCloseVisualState();
              }}
              onClick={() => {
                resetWindowsCloseVisualState();
                void closeWindow();
              }}
            >
              <svg aria-hidden="true" viewBox="0 0 10 10" className="h-[10px] w-[10px]">
                <path d="M1 1L9 9M9 1L1 9" fill="none" stroke="currentColor" strokeWidth="1.2" strokeLinecap="square" />
              </svg>
            </button>
          )}
        </div>
        <CustomScrollArea className={contentHeightClass}>
          <div className="mx-auto max-w-[840px] p-[12px] sm:px-5 sm:py-[10px]">
            {t(fallbackLocale, 'loading')}
            {error && <InlineError message={error} />}
          </div>
        </CustomScrollArea>
      </div>
    );
  }

  const locale = resolveLocale(runtime.effectiveLanguage);
  const isRemindersRoute = location.pathname === '/reminders' || location.pathname === '/';

  return (
    <div className="h-full select-none overflow-hidden">
      <div className={`${dragBarHeightClass} relative select-none [--wails-draggable:drag]`}>
        {isWindows && (
          <button
            type="button"
            className={windowsCloseButtonClass}
            aria-label={t(locale, 'close')}
            title={t(locale, 'close')}
            onPointerEnter={() => {
              setIsWindowsCloseHovered(true);
            }}
            onPointerLeave={() => {
              resetWindowsCloseVisualState();
            }}
            onPointerDown={(event) => {
              if (event.button !== 0) return;
              setIsWindowsClosePressed(true);
            }}
            onPointerUp={() => {
              setIsWindowsClosePressed(false);
            }}
            onPointerCancel={() => {
              resetWindowsCloseVisualState();
            }}
            onClick={() => {
              resetWindowsCloseVisualState();
              void closeWindow();
            }}
          >
            <svg aria-hidden="true" viewBox="0 0 10 10" className="h-[10px] w-[10px]">
              <path d="M1 1L9 9M9 1L1 9" fill="none" stroke="currentColor" strokeWidth="1.2" strokeLinecap="square" />
            </svg>
          </button>
        )}
      </div>
      <CustomScrollArea className={contentHeightClass}>
        <div className="mx-auto max-w-[840px] p-[12px] sm:px-5 sm:py-[10px]">
          <HeroHeader
            locale={locale}
            titleRef={titleRef}
            actions={
              <div className="flex items-center gap-2">
                {isRemindersRoute ? (
                  <button
                    ref={addReminderButtonRef}
                    type="button"
                    aria-label={t(locale, 'addReminder')}
                    title={t(locale, 'addReminder')}
                    onClick={() => {
                      const btn = addReminderButtonRef.current;
                      if (btn) {
                        const rect = btn.getBoundingClientRect();
                        setCreatePanelAnchor({
                          top: Math.round(rect.bottom + 8),
                          right: Math.max(12, Math.round(window.innerWidth - rect.right))
                        });
                      } else {
                        setCreatePanelAnchor(null);
                      }
                      setCreatePanelRequestId((prev) => prev + 1);
                    }}
                    className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-[var(--seg-border)] bg-[var(--seg-bg)] text-[var(--text-primary)] transition-colors hover:bg-[var(--seg-hover-bg)] hover:text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--control-focus-ring)]"
                  >
                    <svg aria-hidden="true" viewBox="0 0 20 20" className="h-3.5 w-3.5">
                      <path d="M10 5v10M5 10h10" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
                    </svg>
                  </button>
                ) : null}

                <div className="inline-flex rounded-full border border-[var(--seg-border)] bg-[var(--seg-bg)] p-1 shadow-[var(--shadow-subtle)]">
                  <NavLink
                    to="/reminders"
                    className={({ isActive }) =>
                      `rounded-full px-3 py-1 text-xs font-medium no-underline transition-colors ${
                        isActive
                          ? 'bg-[linear-gradient(140deg,var(--seg-active),var(--seg-active-strong))] text-white shadow-[var(--shadow-raised)]'
                          : 'text-[var(--seg-text)] hover:bg-[var(--seg-hover-bg)] hover:text-[var(--text-primary)]'
                      }`
                    }
                  >
                    {t(locale, 'navReminders')}
                  </NavLink>
                  <NavLink
                    to="/analytics"
                    className={({ isActive }) =>
                      `rounded-full px-3 py-1 text-xs font-medium no-underline transition-colors ${
                        isActive
                          ? 'bg-[linear-gradient(140deg,var(--seg-active),var(--seg-active-strong))] text-white shadow-[var(--shadow-raised)]'
                          : 'text-[var(--seg-text)] hover:bg-[var(--seg-hover-bg)] hover:text-[var(--text-primary)]'
                      }`
                    }
                  >
                    {t(locale, 'navAnalytics')}
                  </NavLink>
                  <NavLink
                    to="/settings"
                    className={({ isActive }) =>
                      `rounded-full px-3 py-1 text-xs font-medium no-underline transition-colors ${
                        isActive
                          ? 'bg-[linear-gradient(140deg,var(--seg-active),var(--seg-active-strong))] text-white shadow-[var(--shadow-raised)]'
                          : 'text-[var(--seg-text)] hover:bg-[var(--seg-hover-bg)] hover:text-[var(--text-primary)]'
                      }`
                    }
                  >
                    {t(locale, 'navSettings')}
                  </NavLink>
                </div>
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
                  runtimeReminders={runtime.reminders}
                  reminderDrafts={reminderDrafts}
                  createPanelRequestId={createPanelRequestId}
                  createPanelAnchor={createPanelAnchor}
                  onReminderEnabledChange={(id, enabled) => {
                    void applyReminderPatch(id, { enabled });
                  }}
                  onReminderIntervalDraftChange={setReminderIntervalDraft}
                  onReminderIntervalDraftNormalize={normalizeReminderIntervalDraft}
                  onReminderBreakDraftChange={setReminderBreakDraft}
                  onReminderBreakDraftNormalize={normalizeReminderBreakDraft}
                  onReminderDraftCommit={(id, intervalValue, breakValue, intervalUnitSec, breakUnitSec) => {
                    return commitReminderDrafts(id, intervalValue, breakValue, intervalUnitSec, breakUnitSec);
                  }}
                  onReminderEditCancel={(id) => {
                    resetReminderDraftToStored(id);
                  }}
                  onCreateReminder={(name, intervalSec, breakSec, reminderType) =>
                    createReminder(name, intervalSec, breakSec, reminderType)
                  }
                  onReminderDelete={(id) => deleteReminder(id)}
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
      </CustomScrollArea>
    </div>
  );
}
