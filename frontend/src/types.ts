export type ReminderRule = {
  enabled: boolean;
  intervalSec: number;
  breakSec: number;
};

export type Settings = {
  globalEnabled: boolean;
  eye: ReminderRule;
  stand: ReminderRule;
  enforcement: {
    overlayEnabled: boolean;
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
  };
  startup: {
    launchAtLogin: boolean;
  };
};

export type RuntimeState = {
  paused: boolean;
  pauseMode?: string;
  pausedUntil?: string;
  currentSession?: {
    status: string;
    reasons: string[];
    remainingSec: number;
    canSkip: boolean;
  };
  nextEyeInSec: number;
  nextStandInSec: number;
  globalEnabled: boolean;
  timerMode: string;
  currentIdleSec: number;
  overlayEnabled: boolean;
  overlaySkipAllowed: boolean;
  overlayNative: boolean;
};

export type SettingsPatch = Partial<{
  globalEnabled: boolean;
  eye: Partial<ReminderRule>;
  stand: Partial<ReminderRule>;
  enforcement: Partial<Settings['enforcement']>;
  sound: Partial<Settings['sound']>;
  timer: Partial<Settings['timer']>;
  ui: Partial<Settings['ui']>;
  startup: Partial<Settings['startup']>;
}>;
