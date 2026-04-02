import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  checkForUpdates,
  getLaunchAtLogin,
  getNotificationCapability,
  getReminders,
  getSettings,
  openExternalURL,
  setLaunchAtLogin,
  updateSettings
} from '../api';
import type { ReminderConfig, Settings, SettingsPatch, UpdateCheckResult } from '../types';
import { useNotificationState } from './useNotificationState';
import { useReminderManager } from './useReminderManager';
import { nearestOptionValue } from './settings/helpers';

const IDLE_THRESHOLD_OPTIONS = [60, 300, 600, 1800, 3600, 7200] as const;

type UseSettingsOptions = {
  setError: (message: string) => void;
  setBootstrapError: (message: string) => void;
  refreshRuntime: () => Promise<unknown>;
};

export function useSettings({ setError, setBootstrapError, refreshRuntime }: UseSettingsOptions) {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [reminders, setReminders] = useState<ReminderConfig[]>([]);
  const [launchAtLogin, setLaunchAtLoginState] = useState(false);
  const [updateState, setUpdateState] = useState<UpdateCheckResult | null>(null);
  const [isCheckingForUpdates, setIsCheckingForUpdates] = useState(false);

  const {
    setNotificationCapability,
    ensureNotificationReadyForNotifyReminder,
    notificationProductState,
    notificationPromptCode,
    notificationPromptVersion,
    showNotificationSettingsAction,
    showNotificationPrompt,
    refreshNotificationCapabilityFromInteraction,
    openSystemNotificationSettings
  } = useNotificationState({ setError });

  const {
    reminderDrafts,
    applyReminderPatch,
    createReminder,
    deleteReminder,
    setReminderIntervalDraft,
    setReminderBreakDraft,
    normalizeReminderIntervalDraft,
    normalizeReminderBreakDraft,
    commitReminderDrafts,
    resetReminderDraftToStored
  } = useReminderManager({
    reminders,
    setReminders,
    setError,
    refreshRuntime,
    ensureNotificationReadyForNotifyReminder
  });

  const loadSettingsData = useCallback(async (): Promise<void> => {
    try {
      const [next, reminderRows, capability] = await Promise.all([
        getSettings(),
        getReminders(),
        getNotificationCapability().catch(() => null)
      ]);
      setSettings(next);
      setReminders(reminderRows);
      setNotificationCapability(capability);
      setBootstrapError('');

      try {
        const startupState = await getLaunchAtLogin();
        setLaunchAtLoginState(startupState);
      } catch (err) {
        setError(String(err));
      }
    } catch (err) {
      setBootstrapError(String(err));
    }
  }, [setBootstrapError, setError, setNotificationCapability]);

  useEffect(() => {
    void loadSettingsData();
  }, [loadSettingsData]);

  const refreshLaunchAtLoginState = useCallback(
    async (silent = false) => {
      try {
        const actual = await getLaunchAtLogin();
        setLaunchAtLoginState(actual);
      } catch (err) {
        if (!silent) {
          setError(String(err));
        }
      }
    },
    [setError]
  );

  useEffect(() => {
    const refreshWhenVisible = () => {
      if (document.visibilityState !== 'visible') return;
      void refreshLaunchAtLoginState(true);
    };

    window.addEventListener('focus', refreshWhenVisible);
    document.addEventListener('visibilitychange', refreshWhenVisible);

    return () => {
      window.removeEventListener('focus', refreshWhenVisible);
      document.removeEventListener('visibilitychange', refreshWhenVisible);
    };
  }, [refreshLaunchAtLoginState]);

  const applyPatch = useCallback(
    async (patch: SettingsPatch) => {
      if (!settings) return;
      setError('');
      try {
        const next = await updateSettings(patch);
        setSettings(next);
        await refreshRuntime();
      } catch (err) {
        setError(String(err));
      }
    },
    [refreshRuntime, setError, settings]
  );

  const applyLaunchAtLogin = useCallback(
    async (enabled: boolean) => {
      setError('');
      try {
        const actual = await setLaunchAtLogin(enabled);
        setLaunchAtLoginState(actual);
      } catch (err) {
        setError(String(err));
        void refreshLaunchAtLoginState(true);
      }
    },
    [refreshLaunchAtLoginState, setError]
  );

  const idleModeSelectValue = useMemo(() => {
    if (!settings) return 'off';
    if (settings.timer.mode !== 'idle_pause') return 'off';
    return String(nearestOptionValue(settings.timer.idlePauseThresholdSec, IDLE_THRESHOLD_OPTIONS));
  }, [settings]);

  const runUpdateCheck = useCallback(
    async ({ silent }: { silent: boolean }) => {
      setIsCheckingForUpdates(true);
      try {
        const next = await checkForUpdates();
        setUpdateState(next);
        if (!silent && !next.updateAvailable) {
          setError('APP_UP_TO_DATE');
        }
      } catch (err) {
        if (!silent) {
          setError(String(err));
        }
      } finally {
        setIsCheckingForUpdates(false);
      }
    },
    [setError]
  );

  useEffect(() => {
    void runUpdateCheck({ silent: true });
  }, [runUpdateCheck]);

  const openUpdateDownload = useCallback(() => {
    const url = updateState?.selectedAsset?.url || updateState?.releaseUrl;
    if (!url) {
      setError('ERR_UPDATE_DOWNLOAD_URL_MISSING');
      return;
    }
    openExternalURL(url);
  }, [setError, updateState]);

  return {
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
    updateState,
    isCheckingForUpdates,
    checkForUpdates: () => runUpdateCheck({ silent: false }),
    openUpdateDownload,
    notificationProductState,
    notificationPromptCode,
    notificationPromptVersion,
    showNotificationSettingsAction,
    showNotificationPrompt,
    refreshNotificationCapabilityFromInteraction,
    reloadSettingsData: loadSettingsData,
    openSystemNotificationSettings
  };
}
