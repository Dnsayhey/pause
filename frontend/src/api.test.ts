import { afterEach, describe, expect, it, vi } from 'vitest';
import { closeWindow, createReminder, getReminders, quitApp } from './api';
import type { ReminderConfig } from './types';

type TestBackend = {
  GetReminders?: () => Promise<ReminderConfig[] | null | undefined>;
  CreateReminder?: (input: unknown) => Promise<ReminderConfig[] | null | undefined>;
  GetPlatformInfo?: () => Promise<{ os: string; arch: string }>;
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
  };

function setTestWindow(partial: Partial<TestWindow>) {
  Object.assign(globalThis, { window: globalThis });
  Object.assign(window as TestWindow, partial);
}

function clearTestWindow() {
  Object.assign(globalThis, { window: {} });
  delete (window as TestWindow).go;
}

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
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
});

describe('checkForUpdates', () => {
  async function loadUpdateApi(feed: unknown, platform: { os: string; arch: string }) {
    vi.stubEnv('VITE_UPDATES_URL', 'https://updates.example/stable.json');
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => feed
      })
    );
    setTestWindow({
      go: {
        app: {
          App: {
            GetPlatformInfo: async () => platform
          }
        }
      }
    });
    vi.resetModules();
    return import('./api');
  }

  const feed = {
    release: { version: '9.9.9', channel: 'stable', url: null },
    assets: [
      {
        name: 'Pause-macos-arm64.dmg',
        os: 'darwin',
        arch: 'arm64',
        url: 'https://downloads.example/Pause-macos-arm64.dmg'
      },
      {
        name: 'Pause-windows-x64-setup.exe',
        os: 'windows',
        arch: 'amd64',
        url: 'https://downloads.example/Pause-windows-x64-setup.exe'
      }
    ]
  };

  it('selects the asset matching the current platform', async () => {
    const { checkForUpdates } = await loadUpdateApi(feed, { os: 'darwin', arch: 'arm64' });

    const result = await checkForUpdates();

    expect(result.selectedAsset?.name).toBe('Pause-macos-arm64.dmg');
  });

  it('does not fall back to an asset from another platform', async () => {
    const { checkForUpdates } = await loadUpdateApi(
      { release: { version: '9.9.9', channel: 'stable', url: null }, assets: [feed.assets[1]] },
      { os: 'darwin', arch: 'arm64' }
    );

    const result = await checkForUpdates();

    expect(result.selectedAsset).toBeNull();
    expect(result.releaseUrl).toBeNull();
  });

  it('ignores assets whose download url is not http(s)', async () => {
    const { checkForUpdates } = await loadUpdateApi(
      {
        release: { version: '9.9.9', channel: 'stable', url: null },
        assets: [{ ...feed.assets[0], url: 'file:///tmp/Pause.dmg' }]
      },
      { os: 'darwin', arch: 'arm64' }
    );

    const result = await checkForUpdates();

    expect(result.selectedAsset).toBeNull();
  });
});
