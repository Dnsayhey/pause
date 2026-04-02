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
  GetNotificationCapability: () => Promise<NotificationCapability>;
  RequestNotificationPermission: () => Promise<NotificationCapability>;
  OpenNotificationSettings: () => Promise<void>;
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

function detectUpdatePlatform(): { os: string; arch: string } {
  if (typeof navigator === 'undefined') {
    return { os: 'unknown', arch: 'unknown' };
  }
  const nav = navigator as Navigator & { userAgentData?: { platform?: string; architecture?: string } };
  const platform = (nav.userAgentData?.platform || navigator.platform || '').toLowerCase();
  const architectureHint = (nav.userAgentData?.architecture || '').toLowerCase();
  const ua = (navigator.userAgent || '').toLowerCase();

  let os = 'unknown';
  if (platform.includes('mac') || ua.includes('mac os')) os = 'macos';
  else if (platform.includes('win') || ua.includes('windows')) os = 'windows';
  else if (platform.includes('linux') || ua.includes('linux')) os = 'linux';

  let arch = 'unknown';
  if (architectureHint.includes('arm')) arch = 'arm64';
  else if (architectureHint.includes('x86') || architectureHint.includes('x64')) arch = 'x64';
  else if (ua.includes('aarch64') || ua.includes('arm64')) arch = 'arm64';
  else if (ua.includes('x86_64') || ua.includes('win64') || ua.includes('wow64') || ua.includes('intel')) arch = 'x64';

  return { os, arch };
}

function selectBestAsset(assets: UpdateCheckResult['selectedAsset'][] | undefined, os: string, arch: string) {
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

function deriveReleasesPageURL(updatesURL: string): string | null {
  const normalized = String(updatesURL ?? '').trim();
  if (normalized === '') {
    return null;
  }
  try {
    const url = new URL(normalized);
    const parts = url.pathname.split('/').filter(Boolean);
    if (url.hostname.endsWith('.github.io') && parts.length >= 2) {
      return `https://github.com/${url.hostname.replace(/\.github\.io$/i, '')}/${parts[0]}/releases`;
    }
  } catch {
    return null;
  }
  return null;
}

export async function checkForUpdates(): Promise<UpdateCheckResult> {
  if (UPDATES_URL.trim() === '') {
    throw new Error('Update feed URL is not configured.');
  }

  const response = await fetch(UPDATES_URL, {
    cache: 'no-store'
  });
  if (!response.ok) {
    throw new Error(`Failed to fetch update feed: HTTP ${response.status}`);
  }

  const payload = (await response.json()) as UpdatesFeed;
  const latestVersion = String(payload.release?.version ?? '').trim();
  const { os, arch } = detectUpdatePlatform();
  const selectedAsset = selectBestAsset(payload.assets as UpdateAsset[] | undefined, os, arch);

  return {
    currentVersion: APP_VERSION,
    latestVersion: latestVersion || null,
    channel: payload.release?.channel?.trim() || null,
    checkedAt: new Date().toISOString(),
    updateAvailable: latestVersion !== '' && compareVersions(latestVersion, APP_VERSION) > 0,
    releaseUrl: payload.release?.url?.trim() || selectedAsset?.url || null,
    releasesPageUrl: deriveReleasesPageURL(UPDATES_URL),
    selectedAsset
  };
}

export function openExternalURL(url: string): void {
  const target = String(url ?? '').trim();
  if (target === '') {
    throw new Error('URL is required.');
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
