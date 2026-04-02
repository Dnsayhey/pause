import { ToggleSwitchRow } from './ToggleSwitch';
import { PillSelect } from './ui';
import { t, type Locale } from '../i18n';
import type { Settings, SettingsPatch, UpdateCheckResult } from '../types';

type SystemSettingsCardProps = {
  locale: Locale;
  settings: Settings;
  launchAtLogin: boolean;
  idleModeSelectValue: string;
  showTrayCountdownOption: boolean;
  updateState: UpdateCheckResult | null;
  isCheckingForUpdates: boolean;
  onLaunchAtLoginChange: (enabled: boolean) => Promise<void>;
  onPatch: (patch: SettingsPatch) => Promise<void>;
  onCheckForUpdates: () => Promise<void>;
  onOpenUpdateDownload: () => void;
  onThemeLabelDoubleClick?: () => void;
};

export function SystemSettingsCard({
  locale,
  settings,
  launchAtLogin,
  idleModeSelectValue,
  showTrayCountdownOption,
  updateState,
  isCheckingForUpdates,
  onLaunchAtLoginChange,
  onPatch,
  onCheckForUpdates,
  onOpenUpdateDownload,
  onThemeLabelDoubleClick
}: SystemSettingsCardProps) {
  const currentVersion = (updateState?.currentVersion ?? import.meta.env.VITE_APP_VERSION) || '0.0.0';
  const versionDisplay =
    updateState?.updateAvailable && updateState.latestVersion
      ? `v${currentVersion} > v${updateState.latestVersion}`
      : `v${currentVersion}`;

  return (
    <section>
      <h3 className="mb-3 mt-0 text-[18px]">{t(locale, 'sectionSettings')}</h3>
      <div className="grid gap-2.5">
        <ToggleSwitchRow
          label={t(locale, 'launchAtLogin')}
          checked={launchAtLogin}
          onChange={(checked) => {
            void onLaunchAtLoginChange(checked);
          }}
        />
        <ToggleSwitchRow
          label={t(locale, 'overlaySkipAllowed')}
          checked={settings.enforcement.overlaySkipAllowed}
          onChange={(checked) => {
            void onPatch({ enforcement: { overlaySkipAllowed: checked } });
          }}
        />
        <div className="flex flex-col items-start justify-between gap-3 text-sm font-normal leading-[1.35] sm:flex-row sm:items-center">
          <span className="text-[var(--text-primary)]">{t(locale, 'language')}</span>
          <PillSelect
            variant="minimal"
            value={settings.ui.language}
            onChange={(e) => {
              const next = e.target.value as Settings['ui']['language'];
              void onPatch({ ui: { language: next } });
            }}
            options={[
              { value: 'auto', label: t(locale, 'languageAuto') },
              { value: 'zh-CN', label: t(locale, 'languageZhCN') },
              { value: 'en-US', label: t(locale, 'languageEnUS') }
            ]}
          />
        </div>
        <div className="flex flex-col items-start justify-between gap-3 text-sm font-normal leading-[1.35] sm:flex-row sm:items-center">
          <span className="text-[var(--text-primary)]" onDoubleClick={onThemeLabelDoubleClick}>
            {t(locale, 'theme')}
          </span>
          <PillSelect
            variant="minimal"
            value={settings.ui.theme}
            onChange={(e) => {
              const next = e.target.value as Settings['ui']['theme'];
              void onPatch({ ui: { theme: next } });
            }}
            options={[
              { value: 'auto', label: t(locale, 'themeAuto') },
              { value: 'light', label: t(locale, 'themeLight') },
              { value: 'dark', label: t(locale, 'themeDark') }
            ]}
          />
        </div>
        <div className="flex flex-col items-start justify-between gap-3 text-sm font-normal leading-[1.35] sm:flex-row sm:items-center">
          <span className="text-[var(--text-primary)]">{t(locale, 'stopOnIdleEnabled')}</span>
          <PillSelect
            variant="minimal"
            value={idleModeSelectValue}
            onChange={(e) => {
              const next = e.target.value;
              if (next === 'off') {
                void onPatch({ timer: { mode: 'real_time' } });
                return;
              }
              void onPatch({
                timer: {
                  mode: 'idle_pause',
                  idlePauseThresholdSec: Number(next)
                }
              });
            }}
            options={[
              { value: 'off', label: t(locale, 'off') },
              { value: '60', label: t(locale, 'idleOption1Minute') },
              { value: '300', label: t(locale, 'idleOption5Minutes') },
              { value: '600', label: t(locale, 'idleOption10Minutes') },
              { value: '1800', label: t(locale, 'idleOption30Minutes') },
              { value: '3600', label: t(locale, 'idleOption1Hour') },
              { value: '7200', label: t(locale, 'idleOption2Hours') }
            ]}
          />
        </div>
        <ToggleSwitchRow
          label={t(locale, 'endSoundEnabled')}
          checked={settings.sound.enabled}
          onChange={(checked) => {
            void onPatch({ sound: { enabled: checked } });
          }}
        />
        {showTrayCountdownOption && (
          <ToggleSwitchRow
            label={t(locale, 'showTrayCountdown')}
            checked={settings.ui.showTrayCountdown}
            onChange={(checked) => {
              void onPatch({ ui: { showTrayCountdown: checked } });
            }}
          />
        )}
        <div className="flex flex-col items-start justify-between gap-3 text-sm font-normal leading-[1.35] sm:flex-row sm:items-center">
          <span className="text-[var(--text-primary)]">{t(locale, 'updateSectionTitle')}</span>
          <div className="flex flex-wrap items-center justify-end gap-3 self-stretch sm:self-auto">
            <span className="inline-flex items-center gap-1.5 text-xs font-medium text-[var(--text-tertiary)]">
              {updateState?.updateAvailable ? <span className="h-1.5 w-1.5 rounded-full bg-[var(--accent)]" aria-hidden="true" /> : null}
              <span>{versionDisplay}</span>
            </span>
            <button
              type="button"
              className="cursor-pointer border-0 bg-transparent p-0 text-xs font-medium leading-[1.2] text-[var(--text-secondary)] underline decoration-[var(--surface-border-strong)] underline-offset-[3px] transition-colors hover:text-[var(--text-primary)] disabled:cursor-not-allowed disabled:opacity-60"
              disabled={isCheckingForUpdates}
              onClick={() => {
                void onCheckForUpdates();
              }}
            >
              {isCheckingForUpdates ? t(locale, 'updateChecking') : t(locale, 'updateCheckNow')}
            </button>
            {updateState?.updateAvailable && (updateState.selectedAsset?.url || updateState.releaseUrl) ? (
              <button
                type="button"
                className="cursor-pointer border-0 bg-transparent p-0 text-xs font-medium leading-[1.2] text-[var(--text-secondary)] underline decoration-[var(--surface-border-strong)] underline-offset-[3px] transition-colors hover:text-[var(--text-primary)] disabled:cursor-not-allowed disabled:opacity-60"
                onClick={onOpenUpdateDownload}
              >
                {t(locale, 'updateDownload')}
              </button>
            ) : null}
          </div>
        </div>
      </div>
    </section>
  );
}
