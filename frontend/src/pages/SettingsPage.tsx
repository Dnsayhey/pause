import { SystemSettingsCard } from '../components/SystemSettingsCard';
import type { Locale } from '../i18n';
import type { Settings, SettingsPatch } from '../types';

type SettingsPageProps = {
  locale: Locale;
  settings: Settings;
  launchAtLogin: boolean;
  idleModeSelectValue: string;
  soundModeSelectValue: string;
  showTrayCountdownOption: boolean;
  onLaunchAtLoginChange: (enabled: boolean) => Promise<void>;
  onPatch: (patch: SettingsPatch) => Promise<void>;
};

export function SettingsPage({
  locale,
  settings,
  launchAtLogin,
  idleModeSelectValue,
  soundModeSelectValue,
  showTrayCountdownOption,
  onLaunchAtLoginChange,
  onPatch
}: SettingsPageProps) {
  return (
    <section className="mt-3">
      <SystemSettingsCard
        locale={locale}
        settings={settings}
        launchAtLogin={launchAtLogin}
        idleModeSelectValue={idleModeSelectValue}
        soundModeSelectValue={soundModeSelectValue}
        showTrayCountdownOption={showTrayCountdownOption}
        onLaunchAtLoginChange={onLaunchAtLoginChange}
        onPatch={onPatch}
      />
    </section>
  );
}
