import type { ReminderConfig, ReminderPatch, RuntimeState, Settings, SettingsPatch } from './types';

type Backend = {
  GetSettings: () => Promise<Settings>;
  UpdateSettings: (patch: SettingsPatch) => Promise<Settings>;
  GetReminders: () => Promise<ReminderConfig[]>;
  UpdateReminders: (patches: ReminderPatch[]) => Promise<ReminderConfig[]>;
  GetLaunchAtLogin: () => Promise<boolean>;
  SetLaunchAtLogin: (enabled: boolean) => Promise<boolean>;
  GetRuntimeState: () => Promise<RuntimeState>;
  Pause: () => Promise<RuntimeState>;
  Resume: () => Promise<RuntimeState>;
  SkipCurrentBreak: () => Promise<RuntimeState>;
  Quit?: () => Promise<void> | void;
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
