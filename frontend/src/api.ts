import type { RuntimeState, Settings, SettingsPatch } from './types';

type Backend = {
  GetSettings: () => Promise<Settings>;
  UpdateSettings: (patch: SettingsPatch) => Promise<Settings>;
  GetLaunchAtLogin: () => Promise<boolean>;
  SetLaunchAtLogin: (enabled: boolean) => Promise<boolean>;
  GetRuntimeState: () => Promise<RuntimeState>;
  Pause: (mode: string, durationSec: number) => Promise<RuntimeState>;
  Resume: () => Promise<RuntimeState>;
  SkipCurrentBreak: () => Promise<RuntimeState>;
  Quit?: () => Promise<void> | void;
};

const fallbackSettings: Settings = {
  globalEnabled: true,
  eye: { enabled: true, intervalSec: 1200, breakSec: 20 },
  stand: { enabled: true, intervalSec: 3600, breakSec: 300 },
  enforcement: { overlaySkipAllowed: true },
  sound: { enabled: true, volume: 70 },
  timer: { mode: 'idle_pause', idlePauseThresholdSec: 300 },
  ui: { showTrayCountdown: true, language: 'auto', theme: 'auto' }
};

let devSettings = structuredClone(fallbackSettings);
let devLaunchAtLogin = true;
let devRuntime: RuntimeState = {
  paused: false,
  nextEyeInSec: 1200,
  nextStandInSec: 3600,
  globalEnabled: true,
  timerMode: 'idle_pause',
  currentIdleSec: 0,
  overlaySkipAllowed: true,
  overlayNative: false
};

function getBackend(): Backend | null {
  const maybe = (window as unknown as { go?: { main?: { App?: Backend } } }).go;
  return maybe?.main?.App ?? null;
}

export async function getSettings(): Promise<Settings> {
  const backend = getBackend();
  if (!backend) return devSettings;
  return backend.GetSettings();
}

export async function updateSettings(patch: SettingsPatch): Promise<Settings> {
  const backend = getBackend();
  if (!backend) {
    devSettings = {
      ...devSettings,
      ...patch,
      eye: { ...devSettings.eye, ...(patch.eye ?? {}) },
      stand: { ...devSettings.stand, ...(patch.stand ?? {}) },
      enforcement: { ...devSettings.enforcement, ...(patch.enforcement ?? {}) },
      sound: { ...devSettings.sound, ...(patch.sound ?? {}) },
      timer: { ...devSettings.timer, ...(patch.timer ?? {}) },
      ui: { ...devSettings.ui, ...(patch.ui ?? {}) }
    };
    devRuntime.overlaySkipAllowed = devSettings.enforcement.overlaySkipAllowed;
    return devSettings;
  }
  return backend.UpdateSettings(patch);
}

export async function getLaunchAtLogin(): Promise<boolean> {
  const backend = getBackend();
  if (!backend) return devLaunchAtLogin;
  return backend.GetLaunchAtLogin();
}

export async function setLaunchAtLogin(enabled: boolean): Promise<boolean> {
  const backend = getBackend();
  if (!backend) {
    devLaunchAtLogin = enabled;
    return devLaunchAtLogin;
  }
  return backend.SetLaunchAtLogin(enabled);
}

export async function getRuntimeState(): Promise<RuntimeState> {
  const backend = getBackend();
  if (!backend) return devRuntime;
  return backend.GetRuntimeState();
}

export async function pause(mode: string, durationSec = 0): Promise<RuntimeState> {
  const backend = getBackend();
  if (!backend) {
    devRuntime.paused = true;
    devRuntime.pauseMode = mode;
    return devRuntime;
  }
  return backend.Pause(mode, durationSec);
}

export async function resume(): Promise<RuntimeState> {
  const backend = getBackend();
  if (!backend) {
    devRuntime.paused = false;
    devRuntime.pauseMode = '';
    return devRuntime;
  }
  return backend.Resume();
}

export async function skipCurrentBreak(): Promise<RuntimeState> {
  const backend = getBackend();
  if (!backend) {
    devRuntime.currentSession = undefined;
    return devRuntime;
  }
  return backend.SkipCurrentBreak();
}

export async function quitApp(): Promise<void> {
  const backend = getBackend();
  if (backend?.Quit) {
    await backend.Quit();
    return;
  }
  window.close();
}

type RuntimeBridge = {
  EventsOn: (eventName: string, callback: (payload: unknown) => void) => () => void;
};

export function onRuntimeTick(callback: (state: RuntimeState) => void): (() => void) | null {
  const bridge = (window as unknown as { runtime?: RuntimeBridge }).runtime;
  if (!bridge?.EventsOn) return null;
  return bridge.EventsOn('runtime:tick', (payload) => {
    const normalized = Array.isArray(payload) ? payload[0] : payload;
    callback(normalized as RuntimeState);
  });
}

export function onOverlayEvent(callback: (active: boolean) => void): (() => void) | null {
  const bridge = (window as unknown as { runtime?: RuntimeBridge }).runtime;
  if (!bridge?.EventsOn) return null;
  return bridge.EventsOn('break:overlay', (payload) => {
    const normalized = Array.isArray(payload) ? payload[0] : payload;
    const active = Boolean((normalized as { active?: boolean })?.active);
    callback(active);
  });
}
