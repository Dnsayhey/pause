import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type PropsWithChildren
} from 'react';
import { ToastContext, type ToastContextValue, type ToastInput, type ToastRecord } from './toastContext';

let toastCounter = 0;

function nextToastID(): string {
  toastCounter += 1;
  return `toast-${toastCounter}`;
}

export function ToastProvider({ children }: PropsWithChildren) {
  const [toasts, setToasts] = useState<ToastRecord[]>([]);
  const timersRef = useRef<Map<string, number>>(new Map());

  const dismissToast = useCallback((target: string) => {
    const normalized = target.trim();
    if (normalized === '') {
      return;
    }
    const timer = timersRef.current.get(normalized);
    if (timer !== undefined) {
      window.clearTimeout(timer);
      timersRef.current.delete(normalized);
    }
    setToasts((prev) => {
      const removed = prev.filter((toast) => toast.id === normalized || toast.key === normalized);
      for (const toast of removed) {
        toast.onDismiss?.();
      }
      return prev.filter((toast) => toast.id !== normalized && toast.key !== normalized);
    });
  }, []);

  const pushToast = useCallback(
    (input: ToastInput): string => {
      const id = nextToastID();
      const toastKey = input.key?.trim() || undefined;
      const dismissKey = toastKey ?? id;
      const nextToast: ToastRecord = {
        ...input,
        id,
        key: toastKey,
        tone: input.tone ?? 'error',
        durationMs: input.durationMs === undefined ? 5000 : input.durationMs
      };

      const existingTimer = timersRef.current.get(dismissKey);
      if (existingTimer !== undefined) {
        window.clearTimeout(existingTimer);
        timersRef.current.delete(dismissKey);
      }

      setToasts((prev) => {
        const filtered = toastKey ? prev.filter((toast) => toast.key !== toastKey) : prev;
        return [...filtered, nextToast].slice(-4);
      });

      if (nextToast.durationMs !== null) {
        const timeoutID = window.setTimeout(() => {
          dismissToast(dismissKey);
        }, nextToast.durationMs);
        timersRef.current.set(dismissKey, timeoutID);
      }

      return id;
    },
    [dismissToast]
  );

  useEffect(() => {
    const timers = timersRef.current;
    return () => {
      for (const timer of timers.values()) {
        window.clearTimeout(timer);
      }
      timers.clear();
    };
  }, []);

  const value = useMemo<ToastContextValue>(
    () => ({
      pushToast,
      dismissToast
    }),
    [dismissToast, pushToast]
  );

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="pointer-events-none fixed inset-x-3 top-10 z-[90] flex justify-center">
        <div className="flex w-full max-w-[520px] flex-col gap-2">
          {toasts.map((toast) => {
            const showAction = typeof toast.onAction === 'function' && Boolean(toast.actionLabel?.trim());
            const accentClass =
              toast.tone === 'error'
                ? 'border-[var(--error-border)]'
                : 'border-[var(--surface-border-strong)]';
            const iconClass =
              toast.tone === 'error'
                ? 'bg-[var(--error-bg)] text-[var(--error-text)]'
                : 'bg-[var(--accent-soft)] text-[var(--accent)]';

            return (
              <section
                key={toast.id}
                aria-live="polite"
                className={`pointer-events-auto rounded-[14px] border bg-[var(--surface-bg)] px-3 py-2.5 text-[13px] text-[var(--text-primary)] shadow-[var(--surface-shadow)] backdrop-blur ${accentClass}`}
                title={toast.message}
              >
                <div className="flex items-center gap-3">
                  <span
                    aria-hidden="true"
                    className={`inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-full text-[11px] font-semibold ${iconClass}`}
                  >
                    !
                  </span>
                  <div className="min-w-0 flex flex-1 items-center gap-2 overflow-hidden">
                    <p className="m-0 truncate whitespace-nowrap leading-5">{toast.message}</p>
                    {showAction ? (
                      <button
                        type="button"
                        className="inline-flex shrink-0 items-center whitespace-nowrap border-0 bg-transparent p-0 text-xs font-medium text-[var(--text-primary)] underline decoration-[var(--surface-border-strong)] underline-offset-[3px] transition-colors hover:text-[var(--accent)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--control-focus-ring)]"
                        onClick={() => toast.onAction?.()}
                      >
                        {toast.actionLabel}
                      </button>
                    ) : null}
                  </div>
                  <button
                    type="button"
                    aria-label="Dismiss"
                    className="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full border-0 bg-transparent p-0 text-[var(--text-tertiary)] transition-colors hover:bg-[var(--surface-muted)] hover:text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--control-focus-ring)]"
                    onClick={() => dismissToast(toast.key ?? toast.id)}
                  >
                    <svg aria-hidden="true" viewBox="0 0 16 16" className="h-3.5 w-3.5" fill="none">
                      <path d="M4 4l8 8M12 4L4 12" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
                    </svg>
                  </button>
                </div>
              </section>
            );
          })}
        </div>
      </div>
    </ToastContext.Provider>
  );
}
