export type ReminderConfig = {
  id: number;
  name: string;
  reminderType: 'rest' | 'notify';
  enabled: boolean;
  intervalSec: number;
  breakSec: number;
};

export type ReminderPatch = {
  id: number;
  name?: string;
  reminderType?: 'rest' | 'notify';
  enabled?: boolean;
  intervalSec?: number;
  breakSec?: number;
};

export type ReminderCreateInput = {
  name: string;
  intervalSec: number;
  breakSec: number;
  reminderType?: 'rest' | 'notify';
  enabled?: boolean;
};

export type ReminderRuntime = {
  id: number;
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
    reasons: number[];
    remainingSec: number;
    canSkip: boolean;
  };
  reminders: ReminderRuntime[];
  nextBreakReason: number[];
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

export type AnalyticsReminderStat = {
  reminderId: number;
  reminderName: string;
  enabled: boolean;
  reminderType: string;
  triggeredCount: number;
  completedCount: number;
  skippedCount: number;
  canceledCount: number;
  totalActualBreakSec: number;
  avgActualBreakSec: number;
};

export type AnalyticsSummaryStats = {
  totalSessions: number;
  totalCompleted: number;
  totalSkipped: number;
  totalCanceled: number;
  totalActualBreakSec: number;
  avgActualBreakSec: number;
};

export type AnalyticsWeeklyStats = {
  fromSec: number;
  toSec: number;
  reminders: AnalyticsReminderStat[];
  summary: AnalyticsSummaryStats;
};

export type AnalyticsSummary = {
  fromSec: number;
  toSec: number;
  totalSessions: number;
  totalCompleted: number;
  totalSkipped: number;
  totalCanceled: number;
  completionRate: number;
  skipRate: number;
  totalActualBreakSec: number;
  avgActualBreakSec: number;
};

export type AnalyticsTrendPoint = {
  day: string;
  totalSessions: number;
  totalCompleted: number;
  totalSkipped: number;
  totalCanceled: number;
  completionRate: number;
  skipRate: number;
  totalActualBreakSec: number;
  avgActualBreakSec: number;
};

export type AnalyticsTrend = {
  fromSec: number;
  toSec: number;
  points: AnalyticsTrendPoint[];
};

export type AnalyticsBreakTypeDistributionItem = {
  reminderId: number;
  reminderName: string;
  triggeredCount: number;
  completedCount: number;
  skippedCount: number;
  canceledCount: number;
  completionRate: number;
  skipRate: number;
  triggeredShare: number;
  reminderType?: string;
  reminderEnabled: boolean;
};

export type AnalyticsBreakTypeDistribution = {
  fromSec: number;
  toSec: number;
  totalTriggered: number;
  items: AnalyticsBreakTypeDistributionItem[];
};

export type AnalyticsHeatmapMetric = 'skip_rate' | 'completion_rate' | 'triggered_count';

export type AnalyticsHeatmapCell = {
  day: string;
  hour: number;
  triggeredCount: number;
  completedCount: number;
  skippedCount: number;
  canceledCount: number;
  value: number;
};

export type AnalyticsHourlyHeatmap = {
  fromSec: number;
  toSec: number;
  metric: AnalyticsHeatmapMetric;
  cells: AnalyticsHeatmapCell[];
};
