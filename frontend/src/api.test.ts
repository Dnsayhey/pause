import { afterEach, describe, expect, it, vi } from 'vitest';
import { closeWindow, createReminder, getReminders, onRuntimeTick, quitApp } from './api';
import type { ReminderConfig, RuntimeState } from './types';

type TestBackend = {
  GetReminders?: () => Promise<ReminderConfig[] | null | undefined>;
  CreateReminder?: (input: unknown) => Promise<ReminderConfig[] | null | undefined>;
  Quit?: () => Promise<void>;
  CloseWindow?: () => Promise<void>;
};

type TestWindow = Window &
  typeof globalThis & {
    go?: {
      app?: {
        App?: TestBackend;
      };
    };
    runtime?: {
      EventsOn?: (eventName: string, callback: (payload: unknown) => void) => () => void;
    };
  };

function setTestWindow(partial: Partial<TestWindow>) {
  Object.assign(globalThis, { window: globalThis });
  Object.assign(window as TestWindow, partial);
}

function clearTestWindow() {
  Object.assign(globalThis, { window: {} });
  delete (window as TestWindow).go;
  delete (window as TestWindow).runtime;
}

afterEach(() => {
  vi.restoreAllMocks();
  clearTestWindow();
});

describe('api bridge', () => {
  it('normalizes reminder list payloads', async () => {
    setTestWindow({
      go: {
        app: {
          App: {
            GetReminders: vi.fn().mockResolvedValue(null),
            CreateReminder: vi.fn().mockResolvedValue(undefined)
          }
        }
      }
    });

    await expect(getReminders()).resolves.toEqual([]);
    await expect(createReminder({ name: 'Hydrate', intervalSec: 600, breakSec: 60 })).resolves.toEqual([]);
  });

  it('throws a clear error when the backend bridge is missing', async () => {
    clearTestWindow();

    await expect(getReminders()).rejects.toThrow('Pause backend bridge unavailable');
  });

  it('throws a specific error when optional backend functions are missing', async () => {
    setTestWindow({
      go: {
        app: {
          App: {}
        }
      }
    });

    await expect(quitApp()).rejects.toThrow('window.go.app.App.Quit is missing');
    await expect(closeWindow()).rejects.toThrow('window.go.app.App.CloseWindow is missing');
  });

  it('subscribes to runtime ticks and unwraps array payloads', () => {
    const unsubscribe = vi.fn();
    const eventsOn = vi.fn((_eventName: string, callback: (payload: unknown) => void) => {
      callback([{ globalEnabled: true } satisfies Partial<RuntimeState>]);
      return unsubscribe;
    });

    setTestWindow({
      runtime: {
        EventsOn: eventsOn
      }
    });

    const callback = vi.fn();
    const returnedUnsubscribe = onRuntimeTick(callback);

    expect(eventsOn).toHaveBeenCalledWith('runtime:tick', expect.any(Function));
    expect(callback).toHaveBeenCalledWith({ globalEnabled: true });
    expect(returnedUnsubscribe).toBe(unsubscribe);
  });
});
