import { useEffect, useState, type KeyboardEvent } from 'react';
import {
  getRuntimeState,
  getSettings,
  onOverlayEvent,
  skipCurrentBreak,
  updateSettings
} from './api';
import type { RuntimeState, Settings, SettingsPatch } from './types';
import { localizeReason, resolveLocale, t } from './i18n';

function formatSec(totalSec: number, offLabel: string): string {
  if (totalSec < 0) return offLabel;
  const m = Math.floor(totalSec / 60);
  const s = totalSec % 60;
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
}

const EYE_DEFAULT_INTERVAL_MIN = 20;
const EYE_DEFAULT_BREAK_SEC = 20;
const STAND_DEFAULT_INTERVAL_HOUR = 1;
const STAND_DEFAULT_BREAK_MIN = 5;
const IDLE_THRESHOLD_OPTIONS = [60, 300, 600, 1800, 3600, 7200] as const;
const SOUND_VOLUME_OPTIONS = [20, 40, 60, 80, 100] as const;

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

function parseInteger(text: string): number | null {
  const trimmed = text.trim();
  if (trimmed === '') return null;
  const value = Number.parseInt(trimmed, 10);
  if (Number.isNaN(value)) return null;
  return value;
}

function blurOnEnter(event: KeyboardEvent<HTMLInputElement>) {
  if (event.key !== 'Enter') return;
  event.preventDefault();
  event.currentTarget.blur();
}

function nearestOptionValue(value: number, options: readonly number[]): number {
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

type ToggleSwitchRowProps = {
  label: string;
  checked: boolean;
  disabled?: boolean;
  onChange: (checked: boolean) => void;
};

type ToggleSwitchProps = {
  ariaLabel: string;
  checked: boolean;
  disabled?: boolean;
  onChange: (checked: boolean) => void;
};

function ToggleSwitch({ ariaLabel, checked, disabled = false, onChange }: ToggleSwitchProps) {
  return (
    <label className={`pill-switch ${checked ? 'is-on' : ''} ${disabled ? 'is-disabled' : ''}`}>
      <input
        type="checkbox"
        aria-label={ariaLabel}
        checked={checked}
        disabled={disabled}
        onChange={(e) => onChange(e.target.checked)}
      />
      <span className="pill-thumb" />
    </label>
  );
}

function ToggleSwitchRow({ label, checked, disabled = false, onChange }: ToggleSwitchRowProps) {
  return (
    <div className="switch-row">
      <span>{label}</span>
      <ToggleSwitch ariaLabel={label} checked={checked} disabled={disabled} onChange={onChange} />
    </div>
  );
}

export function App() {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [runtime, setRuntime] = useState<RuntimeState | null>(null);
  const [error, setError] = useState('');
  const [eyeIntervalMinDraft, setEyeIntervalMinDraft] = useState(String(EYE_DEFAULT_INTERVAL_MIN));
  const [eyeBreakSecDraft, setEyeBreakSecDraft] = useState(String(EYE_DEFAULT_BREAK_SEC));
  const [standIntervalHourDraft, setStandIntervalHourDraft] = useState(String(STAND_DEFAULT_INTERVAL_HOUR));
  const [standBreakMinDraft, setStandBreakMinDraft] = useState(String(STAND_DEFAULT_BREAK_MIN));

  useEffect(() => {
    const platformClass = detectPlatformClass();
    document.body.dataset.platform = platformClass;
    return () => {
      delete document.body.dataset.platform;
    };
  }, []);

  useEffect(() => {
    let mounted = true;
    let timer: number | null = null;

    const load = async () => {
      try {
        const [s, r] = await Promise.all([getSettings(), getRuntimeState()]);
        if (!mounted) return;
        setSettings(s);
        setRuntime(r);
      } catch (err) {
        if (!mounted) return;
        setError(String(err));
      }
    };

    load();
    const offOverlayEvent = onOverlayEvent((active) => {
      if (!mounted) return;
      if (!active) return;
      document.body.requestFullscreen?.().catch(() => undefined);
    });

    const pollRuntime = async () => {
      const state = await getRuntimeState();
      if (mounted) setRuntime(state);
    };

    const startPolling = () => {
      if (timer !== null) return;
      timer = window.setInterval(() => {
        void pollRuntime();
      }, 1000);
    };

    const stopPolling = () => {
      if (timer === null) return;
      window.clearInterval(timer);
      timer = null;
    };

    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        startPolling();
      } else {
        stopPolling();
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    handleVisibilityChange();

    return () => {
      mounted = false;
      offOverlayEvent?.();
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      stopPolling();
    };
  }, []);

  const applyPatch = async (patch: SettingsPatch) => {
    if (!settings) return;
    setError('');
    try {
      const next = await updateSettings(patch);
      setSettings(next);
      setRuntime(await getRuntimeState());
    } catch (err) {
      setError(String(err));
    }
  };

  useEffect(() => {
    if (!settings) return;
    setEyeIntervalMinDraft(String(Math.max(1, Math.round(settings.eye.intervalSec / 60))));
    setEyeBreakSecDraft(String(settings.eye.breakSec));
    setStandIntervalHourDraft(String(Math.max(1, Math.round(settings.stand.intervalSec / 3600))));
    setStandBreakMinDraft(String(Math.max(1, Math.round(settings.stand.breakSec / 60))));
  }, [settings]);

  if (!settings || !runtime) {
    return (
      <div className="app-root">
        <div className="window-drag-strip" />
        <div className="shell">{t(resolveLocale('auto'), 'loading')}</div>
      </div>
    );
  }

  const locale = resolveLocale(settings.ui.language);

  const overlayActive = Boolean(
    !runtime.overlayNative &&
    runtime.currentSession &&
      runtime.currentSession.status === 'resting' &&
      runtime.overlayEnabled
  );
  const stopOnIdle = settings.timer.mode === 'idle_pause';
  const idleModeSelectValue = stopOnIdle
    ? String(nearestOptionValue(settings.timer.idlePauseThresholdSec, IDLE_THRESHOLD_OPTIONS))
    : 'off';
  const soundModeSelectValue = settings.sound.enabled
    ? String(nearestOptionValue(settings.sound.volume, SOUND_VOLUME_OPTIONS))
    : 'off';

  const commitEyeIntervalDraft = async (raw: string) => {
    const intervalMin = parseInteger(raw);
    const valid = intervalMin !== null && intervalMin >= 1;
    if (!valid) {
      setEyeIntervalMinDraft(String(EYE_DEFAULT_INTERVAL_MIN));
      await applyPatch({
        eye: {
          intervalSec: EYE_DEFAULT_INTERVAL_MIN * 60
        }
      });
      return;
    }
    setEyeIntervalMinDraft(String(intervalMin));
    await applyPatch({
      eye: {
        intervalSec: intervalMin * 60
      }
    });
  };

  const commitEyeBreakDraft = async (raw: string) => {
    const breakSec = parseInteger(raw);
    const valid = breakSec !== null && breakSec >= 10 && breakSec <= 60;
    if (!valid) {
      setEyeBreakSecDraft(String(EYE_DEFAULT_BREAK_SEC));
      await applyPatch({
        eye: {
          breakSec: EYE_DEFAULT_BREAK_SEC
        }
      });
      return;
    }
    setEyeBreakSecDraft(String(breakSec));
    await applyPatch({
      eye: {
        breakSec
      }
    });
  };

  const commitStandIntervalDraft = async (raw: string) => {
    const intervalHour = parseInteger(raw);
    const valid = intervalHour !== null && intervalHour >= 1;
    if (!valid) {
      setStandIntervalHourDraft(String(STAND_DEFAULT_INTERVAL_HOUR));
      await applyPatch({
        stand: {
          intervalSec: STAND_DEFAULT_INTERVAL_HOUR * 3600
        }
      });
      return;
    }
    setStandIntervalHourDraft(String(intervalHour));
    await applyPatch({
      stand: {
        intervalSec: intervalHour * 3600
      }
    });
  };

  const commitStandBreakDraft = async (raw: string) => {
    const breakMin = parseInteger(raw);
    const valid = breakMin !== null && breakMin >= 1 && breakMin <= 10;
    if (!valid) {
      setStandBreakMinDraft(String(STAND_DEFAULT_BREAK_MIN));
      await applyPatch({
        stand: {
          breakSec: STAND_DEFAULT_BREAK_MIN * 60
        }
      });
      return;
    }
    setStandBreakMinDraft(String(breakMin));
    await applyPatch({
      stand: {
        breakSec: breakMin * 60
      }
    });
  };

  return (
    <div className="app-root">
      <div className="window-drag-strip" />
      <div className="shell">
        <header className="hero">
        <div className="hero-main">
          <h1>{t(locale, 'appTitle')}</h1>
          <p>{t(locale, 'appSubtitle')}</p>
        </div>
        <div className="hero-lang">
          <select
            className="lang-select"
            value={settings.ui.language}
            onChange={(e) =>
              applyPatch({
                ui: { language: e.target.value as Settings['ui']['language'] }
              })
            }
          >
            <option value="auto">{t(locale, 'languageAuto')}</option>
            <option value="zh-CN">{t(locale, 'languageZhCN')}</option>
            <option value="en-US">{t(locale, 'languageEnUS')}</option>
          </select>
        </div>
        </header>

        {error && <div className="error">{error}</div>}

        <section className="rules-grid">
        <article className="card">
          <div className="card-title-row">
            <h3>{t(locale, 'eyeReminder')}</h3>
            <ToggleSwitch
              ariaLabel={t(locale, 'enabled')}
              checked={settings.eye.enabled}
              onChange={(checked) => applyPatch({ eye: { enabled: checked } })}
            />
          </div>
          <div className="form-grid">
            <label>
              {t(locale, 'eyeIntervalMin')}
              <input
                className="rule-number-input"
                type="number"
                min={1}
                step={1}
                value={eyeIntervalMinDraft}
                onChange={(e) => setEyeIntervalMinDraft(e.target.value)}
                onKeyDown={blurOnEnter}
                onBlur={(e) => {
                  void commitEyeIntervalDraft(e.currentTarget.value);
                }}
              />
            </label>
            <label>
              {t(locale, 'eyeBreakSec')}
              <input
                className="rule-number-input"
                type="number"
                min={10}
                max={60}
                step={1}
                value={eyeBreakSecDraft}
                onChange={(e) => setEyeBreakSecDraft(e.target.value)}
                onKeyDown={blurOnEnter}
                onBlur={(e) => {
                  void commitEyeBreakDraft(e.currentTarget.value);
                }}
              />
            </label>
          </div>
        </article>

        <article className="card">
          <div className="card-title-row">
            <h3>{t(locale, 'standReminder')}</h3>
            <ToggleSwitch
              ariaLabel={t(locale, 'enabled')}
              checked={settings.stand.enabled}
              onChange={(checked) => applyPatch({ stand: { enabled: checked } })}
            />
          </div>
          <div className="form-grid">
            <label>
              {t(locale, 'standIntervalHour')}
              <input
                className="rule-number-input"
                type="number"
                min={1}
                step={1}
                value={standIntervalHourDraft}
                onChange={(e) => setStandIntervalHourDraft(e.target.value)}
                onKeyDown={blurOnEnter}
                onBlur={(e) => {
                  void commitStandIntervalDraft(e.currentTarget.value);
                }}
              />
            </label>
            <label>
              {t(locale, 'standBreakMin')}
              <input
                className="rule-number-input"
                type="number"
                min={1}
                max={10}
                step={1}
                value={standBreakMinDraft}
                onChange={(e) => setStandBreakMinDraft(e.target.value)}
                onKeyDown={blurOnEnter}
                onBlur={(e) => {
                  void commitStandBreakDraft(e.currentTarget.value);
                }}
              />
            </label>
          </div>
        </article>
        </section>

        <section className="card system-card">
        <h3>{t(locale, 'sectionSettings')}</h3>
        <div className="form-grid system-grid">
          <ToggleSwitchRow
            label={t(locale, 'launchAtLogin')}
            checked={settings.startup.launchAtLogin}
            onChange={(checked) => applyPatch({ startup: { launchAtLogin: checked } })}
          />
          <ToggleSwitchRow
            label={t(locale, 'overlayEnabled')}
            checked={settings.enforcement.overlayEnabled}
            onChange={(checked) => applyPatch({ enforcement: { overlayEnabled: checked } })}
          />
          <ToggleSwitchRow
            label={t(locale, 'overlaySkipAllowed')}
            checked={settings.enforcement.overlaySkipAllowed}
            onChange={(checked) => applyPatch({ enforcement: { overlaySkipAllowed: checked } })}
          />
          <div className="switch-row setting-row">
            <span>{t(locale, 'stopOnIdleEnabled')}</span>
            <select
              className="setting-select"
              value={idleModeSelectValue}
              onChange={(e) => {
                const next = e.target.value;
                if (next === 'off') {
                  void applyPatch({ timer: { mode: 'real_time' } });
                  return;
                }
                void applyPatch({
                  timer: {
                    mode: 'idle_pause',
                    idlePauseThresholdSec: Number(next)
                  }
                });
              }}
            >
              <option value="off">{t(locale, 'off')}</option>
              <option value="60">{t(locale, 'idleOption1Minute')}</option>
              <option value="300">{t(locale, 'idleOption5Minutes')}</option>
              <option value="600">{t(locale, 'idleOption10Minutes')}</option>
              <option value="1800">{t(locale, 'idleOption30Minutes')}</option>
              <option value="3600">{t(locale, 'idleOption1Hour')}</option>
              <option value="7200">{t(locale, 'idleOption2Hours')}</option>
            </select>
          </div>
          <div className="switch-row setting-row">
            <span>{t(locale, 'endSoundEnabled')}</span>
            <select
              className="setting-select"
              value={soundModeSelectValue}
              onChange={(e) => {
                const next = e.target.value;
                if (next === 'off') {
                  void applyPatch({ sound: { enabled: false } });
                  return;
                }
                void applyPatch({
                  sound: {
                    enabled: true,
                    volume: Number(next)
                  }
                });
              }}
            >
              <option value="off">{t(locale, 'off')}</option>
              <option value="20">20%</option>
              <option value="40">40%</option>
              <option value="60">60%</option>
              <option value="80">80%</option>
              <option value="100">100%</option>
            </select>
          </div>
          <ToggleSwitchRow
            label={t(locale, 'showTrayCountdown')}
            checked={settings.ui.showTrayCountdown}
            onChange={(checked) => applyPatch({ ui: { showTrayCountdown: checked } })}
          />
        </div>
        </section>

        {overlayActive && runtime.currentSession && (
          <div className="overlay">
            <div className="overlay-card">
              <h2>{t(locale, 'breakTime')}</h2>
              <p>
                {t(locale, 'reason')}: {runtime.currentSession.reasons.map((reason) => localizeReason(reason, locale)).join(' + ')}
              </p>
              <p>
                {t(locale, 'remaining')}: {formatSec(runtime.currentSession.remainingSec, t(locale, 'off'))}
              </p>
              {runtime.overlaySkipAllowed && runtime.currentSession.canSkip && (
                <button onClick={() => skipCurrentBreak().then(setRuntime).catch((e) => setError(String(e)))}>
                  {t(locale, 'emergencySkip')}
                </button>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
