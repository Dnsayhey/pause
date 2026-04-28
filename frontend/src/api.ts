import type {
  AnalyticsBreakTypeDistribution,
  AnalyticsSummary,
  AnalyticsTrend,
  AnalyticsWeeklyStats,
  NotificationCapability,
  ReminderConfig,
  ReminderCreateInput,
  ReminderPatch,
  RuntimeState,
  Settings,
  SettingsPatch,
  PlatformInfo,
  UpdateAsset,
  UpdateCheckResult
} from './types';

type Backend = {
  GetSettings: () => Promise<Settings>;
  UpdateSettings: (patch: SettingsPatch) => Promise<Settings>;
  GetReminders: () => Promise<ReminderConfig[]>;
  CreateReminder: (input: ReminderCreateInput) => Promise<ReminderConfig[]>;
  DeleteReminder: (reminderID: number) => Promise<ReminderConfig[]>;
  UpdateReminder: (patch: ReminderPatch) => Promise<ReminderConfig[]>;
  GetLaunchAtLogin: () => Promise<boolean>;
  SetLaunchAtLogin: (enabled: boolean) => Promise<boolean>;
  GetRuntimeState: () => Promise<RuntimeState>;
  GetAnalyticsWeeklyStats: (fromSec: number, toSec: number) => Promise<AnalyticsWeeklyStats>;
  GetAnalyticsSummary: (fromSec: number, toSec: number) => Promise<AnalyticsSummary>;
  GetAnalyticsTrendByDay: (fromSec: number, toSec: number) => Promise<AnalyticsTrend>;
  GetAnalyticsBreakTypeDistribution: (fromSec: number, toSec: number) => Promise<AnalyticsBreakTypeDistribution>;
  Pause: () => Promise<RuntimeState>;
  Resume: () => Promise<RuntimeState>;
  SkipCurrentBreak: () => Promise<RuntimeState>;
  PostponeCurrentBreak: () => Promise<RuntimeState>;
  GetNotificationCapability: () => Promise<NotificationCapability>;
  RequestNotificationPermission: () => Promise<NotificationCapability>;
  OpenNotificationSettings: () => Promise<void>;
  GetPlatformInfo?: () => Promise<PlatformInfo>;
  Quit?: () => Promise<void> | void;
  CloseWindow?: () => Promise<void> | void;
};

function getBackend(): Backend | null {
  const maybe = (window as unknown as { go?: { app?: { App?: Backend } } }).go;
  return maybe?.app?.App ?? null;
}

function missingBackendError(): Error {
  return new Error('Pause backend bridge unavailable (window.go.app.App is missing).');
}

function requireBackend(): Backend {
  const backend = getBackend();
  if (!backend) {
    throw missingBackendError();
  }
  return backend;
}

function normalizeReminderConfigs(payload: ReminderConfig[] | null | undefined): ReminderConfig[] {
  return Array.isArray(payload) ? payload : [];
}

export async function getSettings(): Promise<Settings> {
  return requireBackend().GetSettings();
}

export async function updateSettings(patch: SettingsPatch): Promise<Settings> {
  return requireBackend().UpdateSettings(patch);
}

export async function getReminders(): Promise<ReminderConfig[]> {
  return normalizeReminderConfigs(await requireBackend().GetReminders());
}

export async function createReminder(input: ReminderCreateInput): Promise<ReminderConfig[]> {
  return normalizeReminderConfigs(await requireBackend().CreateReminder(input));
}

export async function deleteReminder(reminderID: number): Promise<ReminderConfig[]> {
  return normalizeReminderConfigs(await requireBackend().DeleteReminder(reminderID));
}

export async function updateReminder(patch: ReminderPatch): Promise<ReminderConfig[]> {
  return normalizeReminderConfigs(await requireBackend().UpdateReminder(patch));
}

export async function getLaunchAtLogin(): Promise<boolean> {
  return requireBackend().GetLaunchAtLogin();
}

export async function setLaunchAtLogin(enabled: boolean): Promise<boolean> {
  return requireBackend().SetLaunchAtLogin(enabled);
}

export async function getRuntimeState(): Promise<RuntimeState> {
  return requireBackend().GetRuntimeState();
}

export async function getAnalyticsWeeklyStats(fromSec: number, toSec: number): Promise<AnalyticsWeeklyStats> {
  return requireBackend().GetAnalyticsWeeklyStats(fromSec, toSec);
}

export async function getAnalyticsSummary(fromSec: number, toSec: number): Promise<AnalyticsSummary> {
  return requireBackend().GetAnalyticsSummary(fromSec, toSec);
}

export async function getAnalyticsTrendByDay(fromSec: number, toSec: number): Promise<AnalyticsTrend> {
  return requireBackend().GetAnalyticsTrendByDay(fromSec, toSec);
}

export async function getAnalyticsBreakTypeDistribution(fromSec: number, toSec: number): Promise<AnalyticsBreakTypeDistribution> {
  return requireBackend().GetAnalyticsBreakTypeDistribution(fromSec, toSec);
}

export async function skipCurrentBreak(): Promise<RuntimeState> {
  return requireBackend().SkipCurrentBreak();
}

export async function postponeCurrentBreak(): Promise<RuntimeState> {
  return requireBackend().PostponeCurrentBreak();
}

export async function getNotificationCapability(): Promise<NotificationCapability> {
  return requireBackend().GetNotificationCapability();
}

export async function requestNotificationPermission(): Promise<NotificationCapability> {
  return requireBackend().RequestNotificationPermission();
}

export async function openNotificationSettings(): Promise<void> {
  return requireBackend().OpenNotificationSettings();
}

export async function quitApp(): Promise<void> {
  const backend = requireBackend();
  if (!backend.Quit) {
    throw new Error('Pause backend bridge unavailable (window.go.app.App.Quit is missing).');
  }
  await backend.Quit();
}

export async function closeWindow(): Promise<void> {
  const backend = requireBackend();
  if (!backend.CloseWindow) {
    throw new Error('Pause backend bridge unavailable (window.go.app.App.CloseWindow is missing).');
  }
  await backend.CloseWindow();
}

type RuntimeBrowserBridge = {
  BrowserOpenURL: (url: string) => void;
};

function requireBrowserBridge(): RuntimeBrowserBridge {
  const runtimeBridge = (window as unknown as { runtime?: RuntimeBrowserBridge }).runtime;
  if (!runtimeBridge?.BrowserOpenURL) {
    throw new Error('Pause runtime bridge unavailable (window.runtime.BrowserOpenURL is missing).');
  }
  return runtimeBridge;
}

function normalizeVersion(raw: string): number[] {
  const source = String(raw ?? '').trim().replace(/^v/i, '');
  if (source === '') {
    return [0, 0, 0];
  }
  const [core] = source.split('-', 1);
  const parts = core.split('.').map((part) => Number.parseInt(part, 10));
  return [parts[0] || 0, parts[1] || 0, parts[2] || 0];
}

function compareVersions(left: string, right: string): number {
  const a = normalizeVersion(left);
  const b = normalizeVersion(right);
  for (let index = 0; index < Math.max(a.length, b.length); index += 1) {
    const diff = (a[index] || 0) - (b[index] || 0);
    if (diff !== 0) {
      return diff;
    }
  }
  return 0;
}

function selectBestAsset(assets: UpdateAsset[] | undefined, os: string, arch: string) {
  const rows = Array.isArray(assets) ? assets.filter(Boolean) : [];
  const exact = rows.find((asset) => asset.os === os && asset.arch === arch && typeof asset.url === 'string');
  if (exact) {
    return exact;
  }
  const osOnly = rows.find((asset) => asset.os === os && typeof asset.url === 'string');
  if (osOnly) {
    return osOnly;
  }
  return rows.find((asset) => typeof asset.url === 'string') ?? null;
}

type UpdatesFeed = {
  release?: {
    version?: string;
    channel?: string;
    url?: string | null;
  };
  assets?: Array<{
    name?: string;
    path?: string;
    os?: string;
    arch?: string;
    kind?: string;
    sha256?: string;
    size?: number;
    url?: string | null;
  }>;
};

const APP_VERSION = import.meta.env.VITE_APP_VERSION || '0.0.0';
const UPDATES_URL = import.meta.env.VITE_UPDATES_URL || '';
const UPDATE_FETCH_TIMEOUT_MS = 10_000;

export const ERR_UPDATE_FEED_NOT_CONFIGURED = 'ERR_UPDATE_FEED_NOT_CONFIGURED';
export const ERR_UPDATE_FEED_TIMEOUT = 'ERR_UPDATE_FEED_TIMEOUT';
export const ERR_UPDATE_FEED_HTTP_PREFIX = 'ERR_UPDATE_FEED_HTTP_';
export const ERR_UPDATE_FETCH_FAILED = 'ERR_UPDATE_FETCH_FAILED';
export const ERR_UPDATE_DOWNLOAD_URL_MISSING = 'ERR_UPDATE_DOWNLOAD_URL_MISSING';

async function getPlatformInfo(): Promise<PlatformInfo> {
  const backend = requireBackend();
  if (!backend.GetPlatformInfo) {
    return { os: 'unknown', arch: 'unknown' };
  }
  try {
    return await backend.GetPlatformInfo();
  } catch {
    return { os: 'unknown', arch: 'unknown' };
  }
}

export async function checkForUpdates(): Promise<UpdateCheckResult> {
  if (UPDATES_URL.trim() === '') {
    throw new Error(ERR_UPDATE_FEED_NOT_CONFIGURED);
  }

  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), UPDATE_FETCH_TIMEOUT_MS);
  let response: Response;
  try {
    response = await fetch(UPDATES_URL, {
      cache: 'no-store',
      signal: controller.signal
    });
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') {
      throw new Error(ERR_UPDATE_FEED_TIMEOUT);
    }
    throw new Error(ERR_UPDATE_FETCH_FAILED);
  } finally {
    window.clearTimeout(timeout);
  }
  if (!response.ok) {
    throw new Error(`${ERR_UPDATE_FEED_HTTP_PREFIX}${response.status}`);
  }

  let payload: UpdatesFeed;
  try {
    payload = (await response.json()) as UpdatesFeed;
  } catch {
    throw new Error(ERR_UPDATE_FETCH_FAILED);
  }
  const latestVersion = String(payload.release?.version ?? '').trim();
  const { os, arch } = await getPlatformInfo();
  const selectedAsset = selectBestAsset(payload.assets as UpdateAsset[] | undefined, os, arch);

  return {
    currentVersion: APP_VERSION,
    latestVersion: latestVersion || null,
    channel: payload.release?.channel?.trim() || null,
    checkedAt: new Date().toISOString(),
    updateAvailable: latestVersion !== '' && compareVersions(latestVersion, APP_VERSION) > 0,
    releaseUrl: payload.release?.url?.trim() || selectedAsset?.url || null,
    selectedAsset
  };
}

export function openExternalURL(url: string): void {
  const target = String(url ?? '').trim();
  if (target === '') {
    throw new Error(ERR_UPDATE_DOWNLOAD_URL_MISSING);
  }
  requireBrowserBridge().BrowserOpenURL(target);
}

type RuntimeBridge = {
  EventsOn: (eventName: string, callback: (payload: unknown) => void) => () => void;
};

function requireRuntimeBridge(): RuntimeBridge {
  const bridge = (window as unknown as { runtime?: RuntimeBridge }).runtime;
  if (!bridge?.EventsOn) {
    throw new Error('Pause runtime bridge unavailable (window.runtime.EventsOn is missing).');
  }
  return bridge;
}

export function onRuntimeTick(callback: (state: RuntimeState) => void): () => void {
  const bridge = requireRuntimeBridge();
  return bridge.EventsOn('runtime:tick', (payload) => {
    const normalized = Array.isArray(payload) ? payload[0] : payload;
    callback(normalized as RuntimeState);
  });
}
