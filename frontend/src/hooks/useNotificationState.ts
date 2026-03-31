import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  getNotificationCapability,
  openNotificationSettings as openNotificationSettingsAPI,
  requestNotificationPermission as requestNotificationPermissionAPI
} from '../api';
import type { NotificationCapability, NotificationProductState } from '../types';
import {
  notificationErrorCodeFromCapability,
  notificationProductStateFromCapability
} from './settings/helpers';

type UseNotificationStateOptions = {
  setError: (message: string) => void;
};

export function useNotificationState({ setError }: UseNotificationStateOptions) {
  const [notificationCapability, setNotificationCapability] = useState<NotificationCapability | null>(null);
  const [notificationPromptCode, setNotificationPromptCode] = useState('');
  const [notificationPromptVersion, setNotificationPromptVersion] = useState(0);
  const [showNotificationSettingsAction, setShowNotificationSettingsAction] = useState(false);
  const notificationCapabilityInteractionRefreshInFlightRef = useRef(false);

  const clearNotificationPrompt = useCallback(() => {
    setNotificationPromptCode('');
    setShowNotificationSettingsAction(false);
  }, []);

  const applyNotificationPrompt = useCallback(
    (capability: NotificationCapability | null) => {
      const code = notificationErrorCodeFromCapability(capability);
      if (code === '') {
        clearNotificationPrompt();
        return;
      }
      setNotificationPromptCode(code);
      setNotificationPromptVersion((prev) => prev + 1);
      const productState = notificationProductStateFromCapability(capability);
      setShowNotificationSettingsAction(productState === 'unavailable' && Boolean(capability?.canOpenSettings));
    },
    [clearNotificationPrompt]
  );

  const refreshNotificationCapability = useCallback(
    async (silent = false): Promise<NotificationCapability | null> => {
      try {
        const capability = await getNotificationCapability();
        setNotificationCapability(capability);
        return capability;
      } catch (err) {
        if (!silent) {
          setError(String(err));
        }
        return null;
      }
    },
    [setError]
  );

  const ensureNotificationReadyForNotifyReminder = useCallback(
    async (): Promise<boolean> => {
      const capability = await refreshNotificationCapability(false);
      if (!capability) {
        return false;
      }

      const productState = notificationProductStateFromCapability(capability);
      if (productState === 'available') {
        clearNotificationPrompt();
        return true;
      }

      if (productState === 'pending') {
        applyNotificationPrompt(capability);
        void requestNotificationPermissionAPI()
          .then((requested) => {
            if (notificationProductStateFromCapability(requested) === 'available') {
              setNotificationCapability(requested);
              clearNotificationPrompt();
            }
          })
          .catch((err) => {
            setError(String(err));
          });
        return false;
      }

      applyNotificationPrompt(capability);
      return false;
    },
    [applyNotificationPrompt, clearNotificationPrompt, refreshNotificationCapability, setError]
  );

  const refreshNotificationCapabilityFromInteraction = useCallback(() => {
    if (notificationCapabilityInteractionRefreshInFlightRef.current) {
      return;
    }
    notificationCapabilityInteractionRefreshInFlightRef.current = true;
    void refreshNotificationCapability(true)
      .then((capability) => {
        if (notificationProductStateFromCapability(capability) === 'available') {
          clearNotificationPrompt();
        }
      })
      .finally(() => {
        notificationCapabilityInteractionRefreshInFlightRef.current = false;
      });
  }, [clearNotificationPrompt, refreshNotificationCapability]);

  const showNotificationPrompt = useCallback(
    (state: Exclude<NotificationProductState, 'available'>) => {
      applyNotificationPrompt(notificationCapability);
      if (state !== 'pending') {
        return;
      }
      void requestNotificationPermissionAPI()
        .then((requested) => {
          if (notificationProductStateFromCapability(requested) === 'available') {
            setNotificationCapability(requested);
            clearNotificationPrompt();
          }
        })
        .catch((err) => {
          setError(String(err));
        });
    },
    [applyNotificationPrompt, clearNotificationPrompt, notificationCapability, setError]
  );

  const notificationProductState = useMemo(
    () => notificationProductStateFromCapability(notificationCapability),
    [notificationCapability]
  );

  const openSystemNotificationSettings = useCallback(async () => {
    try {
      await openNotificationSettingsAPI();
    } catch (err) {
      setError(String(err));
    }
  }, [setError]);

  useEffect(() => {
    const refreshWhenVisible = () => {
      if (document.visibilityState !== 'visible') return;
      void refreshNotificationCapability(true);
    };

    window.addEventListener('focus', refreshWhenVisible);
    document.addEventListener('visibilitychange', refreshWhenVisible);

    return () => {
      window.removeEventListener('focus', refreshWhenVisible);
      document.removeEventListener('visibilitychange', refreshWhenVisible);
    };
  }, [refreshNotificationCapability]);

  return {
    notificationCapability,
    setNotificationCapability,
    refreshNotificationCapability,
    ensureNotificationReadyForNotifyReminder,
    notificationProductState,
    notificationPromptCode,
    notificationPromptVersion,
    showNotificationSettingsAction,
    showNotificationPrompt,
    refreshNotificationCapabilityFromInteraction,
    openSystemNotificationSettings
  };
}
