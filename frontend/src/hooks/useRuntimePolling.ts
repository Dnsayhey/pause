import { useCallback, useEffect, useRef, useState } from 'react';
import { getRuntimeState } from '../api';
import type { RuntimeState } from '../types';

type UseRuntimePollingOptions = {
  setError: (message: string) => void;
  setBootstrapError: (message: string) => void;
  clearError: () => void;
};

export function useRuntimePolling({ setError, setBootstrapError, clearError }: UseRuntimePollingOptions) {
  const [runtime, setRuntime] = useState<RuntimeState | null>(null);
  const mountedRef = useRef(false);
  const hasLoadedRuntimeRef = useRef(false);
  const lastReportedErrorRef = useRef('');

  const refreshRuntime = useCallback(async (): Promise<RuntimeState | null> => {
    try {
      const state = await getRuntimeState();
      if (mountedRef.current) {
        setRuntime(state);
        setBootstrapError('');
        if (lastReportedErrorRef.current !== '') {
          lastReportedErrorRef.current = '';
          clearError();
        }
      }
      hasLoadedRuntimeRef.current = true;
      return state;
    } catch (err) {
      const message = String(err);
      if (mountedRef.current) {
        if (!hasLoadedRuntimeRef.current) {
          setBootstrapError(message);
        } else if (lastReportedErrorRef.current !== message) {
          lastReportedErrorRef.current = message;
          setError(message);
        }
      }
      return null;
    }
  }, [clearError, setBootstrapError, setError]);

  useEffect(() => {
    mountedRef.current = true;
    let timer: number | null = null;

    const startPolling = () => {
      if (timer !== null) return;
      timer = window.setInterval(() => {
        void refreshRuntime();
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

    void refreshRuntime();
    document.addEventListener('visibilitychange', handleVisibilityChange);
    handleVisibilityChange();

    return () => {
      mountedRef.current = false;
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      stopPolling();
    };
  }, [refreshRuntime]);

  return {
    runtime,
    setRuntime,
    refreshRuntime,
    resetReportedError: () => {
      lastReportedErrorRef.current = '';
    }
  };
}
