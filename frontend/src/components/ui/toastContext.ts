import { createContext } from 'react';

export type ToastTone = 'error' | 'info';

export type ToastInput = {
  key?: string;
  message: string;
  tone?: ToastTone;
  actionLabel?: string;
  onAction?: () => void;
  onDismiss?: () => void;
  durationMs?: number | null;
};

export type ToastRecord = ToastInput & {
  id: string;
  tone: ToastTone;
  durationMs: number | null;
};

export type ToastContextValue = {
  pushToast: (input: ToastInput) => string;
  dismissToast: (target: string) => void;
};

export const ToastContext = createContext<ToastContextValue | null>(null);
