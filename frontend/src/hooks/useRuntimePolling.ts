import { useCallback, useEffect, useRef, useState } from 'react';
import { getRuntimeState, onOverlayEvent } from '../api';
import type { RuntimeState } from '../types';

type UseRuntimePollingOptions = {
  setError: (message: string) => void;
};

export function useRuntimePolling({ setError }: UseRuntimePollingOptions) {
  const [runtime, setRuntime] = useState<RuntimeState | null>(null);
  const mountedRef = useRef(false);

  const refreshRuntime = useCallback(async (): Promise<RuntimeState | null> => {
    try {
      const state = await getRuntimeState();
      if (mountedRef.current) {
        setRuntime(state);
      }
      return state;
    } catch (err) {
      if (mountedRef.current) {
        setError(String(err));
      }
      return null;
    }
  }, [setError]);

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

    const offOverlayEvent = onOverlayEvent((active) => {
      if (!mountedRef.current) return;
      if (!active) return;
      document.body.requestFullscreen?.().catch(() => undefined);
    });

    void refreshRuntime();
    document.addEventListener('visibilitychange', handleVisibilityChange);
    handleVisibilityChange();

    return () => {
      mountedRef.current = false;
      offOverlayEvent?.();
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      stopPolling();
    };
  }, [refreshRuntime]);

  return {
    runtime,
    setRuntime,
    refreshRuntime
  };
}
