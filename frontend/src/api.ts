import type {
  AnalyticsBreakTypeDistribution,
  AnalyticsHeatmapMetric,
  AnalyticsHourlyHeatmap,
  AnalyticsSummary,
  AnalyticsTrend,
  AnalyticsWeeklyStats,
  ReminderConfig,
  ReminderCreateInput,
  ReminderPatch,
  RuntimeState,
  Settings,
  SettingsPatch
} from './types';

type Backend = {
  GetSettings: () => Promise<Settings>;
  UpdateSettings: (patch: SettingsPatch) => Promise<Settings>;
  GetReminders: () => Promise<ReminderConfig[]>;
  CreateReminder: (input: ReminderCreateInput) => Promise<ReminderConfig[]>;
  DeleteReminder: (reminderID: number) => Promise<ReminderConfig[]>;
  UpdateReminders: (patches: ReminderPatch[]) => Promise<ReminderConfig[]>;
  GetLaunchAtLogin: () => Promise<boolean>;
  SetLaunchAtLogin: (enabled: boolean) => Promise<boolean>;
  GetRuntimeState: () => Promise<RuntimeState>;
  GetAnalyticsWeeklyStats: (fromSec: number, toSec: number) => Promise<AnalyticsWeeklyStats>;
  GetAnalyticsSummary: (fromSec: number, toSec: number) => Promise<AnalyticsSummary>;
  GetAnalyticsTrendByDay: (fromSec: number, toSec: number) => Promise<AnalyticsTrend>;
  GetAnalyticsBreakTypeDistribution: (fromSec: number, toSec: number) => Promise<AnalyticsBreakTypeDistribution>;
  GetAnalyticsHourlyHeatmap: (fromSec: number, toSec: number, metric: string) => Promise<AnalyticsHourlyHeatmap>;
  Pause: () => Promise<RuntimeState>;
  Resume: () => Promise<RuntimeState>;
  SkipCurrentBreak: () => Promise<RuntimeState>;
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

export async function getSettings(): Promise<Settings> {
  return requireBackend().GetSettings();
}

export async function updateSettings(patch: SettingsPatch): Promise<Settings> {
  return requireBackend().UpdateSettings(patch);
}

export async function getReminders(): Promise<ReminderConfig[]> {
  return requireBackend().GetReminders();
}

export async function createReminder(input: ReminderCreateInput): Promise<ReminderConfig[]> {
  return requireBackend().CreateReminder(input);
}

export async function deleteReminder(reminderID: number): Promise<ReminderConfig[]> {
  return requireBackend().DeleteReminder(reminderID);
}

export async function updateReminders(patches: ReminderPatch[]): Promise<ReminderConfig[]> {
  return requireBackend().UpdateReminders(patches);
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

export async function getAnalyticsHourlyHeatmap(
  fromSec: number,
  toSec: number,
  metric: AnalyticsHeatmapMetric = 'skip_rate'
): Promise<AnalyticsHourlyHeatmap> {
  return requireBackend().GetAnalyticsHourlyHeatmap(fromSec, toSec, metric);
}

export async function skipCurrentBreak(): Promise<RuntimeState> {
  return requireBackend().SkipCurrentBreak();
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
