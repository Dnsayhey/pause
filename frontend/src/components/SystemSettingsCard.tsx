import { ToggleSwitchRow } from './ToggleSwitch';
import { GlassCard, PillSelect } from './ui';
import { t, type Locale } from '../i18n';
import type { Settings, SettingsPatch } from '../types';

type SystemSettingsCardProps = {
  locale: Locale;
  settings: Settings;
  launchAtLogin: boolean;
  idleModeSelectValue: string;
  soundModeSelectValue: string;
  showTrayCountdownOption: boolean;
  onLaunchAtLoginChange: (enabled: boolean) => Promise<void>;
  onPatch: (patch: SettingsPatch) => Promise<void>;
};

export function SystemSettingsCard({
  locale,
  settings,
  launchAtLogin,
  idleModeSelectValue,
  soundModeSelectValue,
  showTrayCountdownOption,
  onLaunchAtLoginChange,
  onPatch
}: SystemSettingsCardProps) {
  return (
    <GlassCard>
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
          <span className="text-[var(--text-primary)]">{t(locale, 'theme')}</span>
          <PillSelect
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
        <div className="flex flex-col items-start justify-between gap-3 text-sm font-normal leading-[1.35] sm:flex-row sm:items-center">
          <span className="text-[var(--text-primary)]">{t(locale, 'endSoundEnabled')}</span>
          <PillSelect
            value={soundModeSelectValue}
            onChange={(e) => {
              const next = e.target.value;
              if (next === 'off') {
                void onPatch({ sound: { enabled: false } });
                return;
              }
              void onPatch({
                sound: {
                  enabled: true,
                  volume: Number(next)
                }
              });
            }}
            options={[
              { value: 'off', label: t(locale, 'off') },
              { value: '20', label: '20%' },
              { value: '40', label: '40%' },
              { value: '60', label: '60%' },
              { value: '80', label: '80%' },
              { value: '100', label: '100%' }
            ]}
          />
        </div>
        {showTrayCountdownOption && (
          <ToggleSwitchRow
            label={t(locale, 'showTrayCountdown')}
            checked={settings.ui.showTrayCountdown}
            onChange={(checked) => {
              void onPatch({ ui: { showTrayCountdown: checked } });
            }}
          />
        )}
      </div>
    </GlassCard>
  );
}
