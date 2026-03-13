import { ToggleSwitchRow } from './ToggleSwitch';
import { t, type Locale } from '../i18n';
import type { Settings, SettingsPatch } from '../types';

type SystemSettingsCardProps = {
  locale: Locale;
  settings: Settings;
  launchAtLogin: boolean;
  idleModeSelectValue: string;
  soundModeSelectValue: string;
  onLaunchAtLoginChange: (enabled: boolean) => Promise<void>;
  onPatch: (patch: SettingsPatch) => Promise<void>;
};

export function SystemSettingsCard({
  locale,
  settings,
  launchAtLogin,
  idleModeSelectValue,
  soundModeSelectValue,
  onLaunchAtLoginChange,
  onPatch
}: SystemSettingsCardProps) {
  return (
    <section className="card system-card">
      <h3>{t(locale, 'sectionSettings')}</h3>
      <div className="form-grid system-grid">
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
        <div className="switch-row setting-row">
          <span>{t(locale, 'stopOnIdleEnabled')}</span>
          <select
            className="setting-select"
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
          onChange={(checked) => {
            void onPatch({ ui: { showTrayCountdown: checked } });
          }}
        />
      </div>
    </section>
  );
}
