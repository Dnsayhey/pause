import { useContext } from 'react';
import { ToastContext, type ToastContextValue } from './toastContext';

export function useToast(): ToastContextValue {
  const value = useContext(ToastContext);
  if (!value) {
    throw new Error('useToast must be used within a ToastProvider');
  }
  return value;
}
