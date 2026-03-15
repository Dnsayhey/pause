export type ReminderConfig = {
  id: string;
  name: string;
  deliveryType: 'overlay' | 'notification';
  enabled: boolean;
  intervalSec: number;
  breakSec: number;
};

export type ReminderPatch = {
  id: string;
  name?: string;
  deliveryType?: 'overlay' | 'notification';
  enabled?: boolean;
  intervalSec?: number;
  breakSec?: number;
};

export type ReminderRuntime = {
  id: string;
  enabled: boolean;
  paused: boolean;
  nextInSec: number;
  intervalSec: number;
  breakSec: number;
};

export type Settings = {
  globalEnabled: boolean;
  enforcement: {
    overlaySkipAllowed: boolean;
  };
  sound: {
    enabled: boolean;
    volume: number;
  };
  timer: {
    mode: 'idle_pause' | 'real_time';
    idlePauseThresholdSec: number;
  };
  ui: {
    showTrayCountdown: boolean;
    language: 'auto' | 'zh-CN' | 'en-US';
    theme: 'auto' | 'light' | 'dark';
  };
};

export type RuntimeState = {
  currentSession?: {
    status: string;
    reasons: string[];
    remainingSec: number;
    canSkip: boolean;
  };
  reminders: ReminderRuntime[];
  nextBreakReason: string[];
  globalEnabled: boolean;
  timerMode: string;
  idleThresholdSec: number;
  lastTickActive: boolean;
  showTrayCountdown: boolean;
  currentIdleSec: number;
  overlaySkipAllowed: boolean;
  overlayNative: boolean;
  effectiveLanguage?: 'zh-CN' | 'en-US';
  effectiveTheme?: 'light' | 'dark';
};

export type SettingsPatch = Partial<{
  globalEnabled: boolean;
  enforcement: Partial<Settings['enforcement']>;
  sound: Partial<Settings['sound']>;
  timer: Partial<Settings['timer']>;
  ui: Partial<Settings['ui']>;
}>;
