import { SystemSettingsCard } from '../components/SystemSettingsCard';
import type { Locale } from '../i18n';
import type { Settings, SettingsPatch, UpdateCheckResult } from '../types';

type SettingsPageProps = {
  locale: Locale;
  settings: Settings;
  launchAtLogin: boolean;
  idleModeSelectValue: string;
  soundModeSelectValue: string;
  showTrayCountdownOption: boolean;
  updateState: UpdateCheckResult | null;
  isCheckingForUpdates: boolean;
  onLaunchAtLoginChange: (enabled: boolean) => Promise<void>;
  onPatch: (patch: SettingsPatch) => Promise<void>;
  onCheckForUpdates: () => Promise<void>;
  onOpenUpdateDownload: () => void;
  onThemeLabelDoubleClick?: () => void;
};

export function SettingsPage({
  locale,
  settings,
  launchAtLogin,
  idleModeSelectValue,
  soundModeSelectValue,
  showTrayCountdownOption,
  updateState,
  isCheckingForUpdates,
  onLaunchAtLoginChange,
  onPatch,
  onCheckForUpdates,
  onOpenUpdateDownload,
  onThemeLabelDoubleClick
}: SettingsPageProps) {
  return (
    <section className="mt-3 px-2 sm:px-3">
      <SystemSettingsCard
        locale={locale}
        settings={settings}
        launchAtLogin={launchAtLogin}
        idleModeSelectValue={idleModeSelectValue}
        soundModeSelectValue={soundModeSelectValue}
        showTrayCountdownOption={showTrayCountdownOption}
        updateState={updateState}
        isCheckingForUpdates={isCheckingForUpdates}
        onLaunchAtLoginChange={onLaunchAtLoginChange}
        onPatch={onPatch}
        onCheckForUpdates={onCheckForUpdates}
        onOpenUpdateDownload={onOpenUpdateDownload}
        onThemeLabelDoubleClick={onThemeLabelDoubleClick}
      />
    </section>
  );
}
