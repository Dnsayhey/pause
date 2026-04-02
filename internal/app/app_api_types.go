package app

import "time"

type ReminderConfig struct {
	ID           int64  `json:"id"`
	Name         string `json:"name,omitempty"`
	Enabled      bool   `json:"enabled"`
	IntervalSec  int    `json:"intervalSec"`
	BreakSec     int    `json:"breakSec"`
	ReminderType string `json:"reminderType,omitempty"`
}

type ReminderPatch struct {
	ID           int64   `json:"id"`
	Name         *string `json:"name,omitempty"`
	Enabled      *bool   `json:"enabled,omitempty"`
	IntervalSec  *int    `json:"intervalSec,omitempty"`
	BreakSec     *int    `json:"breakSec,omitempty"`
	ReminderType *string `json:"reminderType,omitempty"`
}

type ReminderCreateInput struct {
	Name         string  `json:"name"`
	IntervalSec  int     `json:"intervalSec"`
	BreakSec     int     `json:"breakSec"`
	Enabled      *bool   `json:"enabled,omitempty"`
	ReminderType *string `json:"reminderType,omitempty"`
}

type EnforcementSettings struct {
	OverlaySkipAllowed bool `json:"overlaySkipAllowed"`
}

type SoundSettings struct {
	Enabled bool `json:"enabled"`
	Volume  int  `json:"volume"`
}

type TimerSettings struct {
	Mode                  string `json:"mode"`
	IdlePauseThresholdSec int    `json:"idlePauseThresholdSec"`
}

type UISettings struct {
	ShowTrayCountdown bool   `json:"showTrayCountdown"`
	Language          string `json:"language"`
	Theme             string `json:"theme"`
}

type Settings struct {
	Enforcement EnforcementSettings `json:"enforcement"`
	Sound       SoundSettings       `json:"sound"`
	Timer       TimerSettings       `json:"timer"`
	UI          UISettings          `json:"ui"`
}

type EnforcementSettingsPatch struct {
	OverlaySkipAllowed *bool `json:"overlaySkipAllowed,omitempty"`
}

type SoundSettingsPatch struct {
	Enabled *bool `json:"enabled,omitempty"`
	Volume  *int  `json:"volume,omitempty"`
}

type TimerSettingsPatch struct {
	Mode                  *string `json:"mode,omitempty"`
	IdlePauseThresholdSec *int    `json:"idlePauseThresholdSec,omitempty"`
}

type UISettingsPatch struct {
	ShowTrayCountdown *bool   `json:"showTrayCountdown,omitempty"`
	Language          *string `json:"language,omitempty"`
	Theme             *string `json:"theme,omitempty"`
}

type SettingsPatch struct {
	Enforcement *EnforcementSettingsPatch `json:"enforcement,omitempty"`
	Sound       *SoundSettingsPatch       `json:"sound,omitempty"`
	Timer       *TimerSettingsPatch       `json:"timer,omitempty"`
	UI          *UISettingsPatch          `json:"ui,omitempty"`
}

type ReminderRuntime struct {
	ID           int64  `json:"id"`
	Name         string `json:"name,omitempty"`
	ReminderType string `json:"reminderType,omitempty"`
	Enabled      bool   `json:"enabled"`
	Paused       bool   `json:"paused"`
	NextInSec    int    `json:"nextInSec"`
	IntervalSec  int    `json:"intervalSec"`
	BreakSec     int    `json:"breakSec"`
}

type BreakSessionView struct {
	Status       string    `json:"status"`
	Reasons      []int64   `json:"reasons"`
	StartedAt    time.Time `json:"startedAt"`
	EndsAt       time.Time `json:"endsAt"`
	RemainingSec int       `json:"remainingSec"`
	CanSkip      bool      `json:"canSkip"`
}

type RuntimeState struct {
	Now                time.Time         `json:"now"`
	CurrentSession     *BreakSessionView `json:"currentSession,omitempty"`
	Reminders          []ReminderRuntime `json:"reminders"`
	NextBreakReason    []int64           `json:"nextBreakReason"`
	GlobalEnabled      bool              `json:"globalEnabled"`
	TimerMode          string            `json:"timerMode"`
	IdleThresholdSec   int               `json:"idleThresholdSec"`
	LastTickActive     bool              `json:"lastTickActive"`
	CurrentIdleSec     int               `json:"currentIdleSec"`
	ShowTrayCountdown  bool              `json:"showTrayCountdown"`
	OverlaySkipAllowed bool              `json:"overlaySkipAllowed"`
	OverlayNative      bool              `json:"overlayNative"`
	EffectiveLanguage  string            `json:"effectiveLanguage,omitempty"`
	EffectiveTheme     string            `json:"effectiveTheme,omitempty"`
}

type NotificationCapability struct {
	PermissionState string `json:"permissionState"`
	CanRequest      bool   `json:"canRequest"`
	CanOpenSettings bool   `json:"canOpenSettings"`
	Reason          string `json:"reason,omitempty"`
}

type PlatformInfo struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

type AnalyticsReminderStat struct {
	ReminderID          int64   `json:"reminderId"`
	ReminderName        string  `json:"reminderName"`
	Enabled             bool    `json:"enabled"`
	ReminderType        string  `json:"reminderType"`
	TriggeredCount      int     `json:"triggeredCount"`
	CompletedCount      int     `json:"completedCount"`
	SkippedCount        int     `json:"skippedCount"`
	CanceledCount       int     `json:"canceledCount"`
	TotalActualBreakSec int     `json:"totalActualBreakSec"`
	AvgActualBreakSec   float64 `json:"avgActualBreakSec"`
}

type AnalyticsSummaryStats struct {
	TotalSessions       int     `json:"totalSessions"`
	TotalCompleted      int     `json:"totalCompleted"`
	TotalSkipped        int     `json:"totalSkipped"`
	TotalCanceled       int     `json:"totalCanceled"`
	TotalActualBreakSec int     `json:"totalActualBreakSec"`
	AvgActualBreakSec   float64 `json:"avgActualBreakSec"`
}

type AnalyticsWeeklyStats struct {
	FromSec   int64                   `json:"fromSec"`
	ToSec     int64                   `json:"toSec"`
	Reminders []AnalyticsReminderStat `json:"reminders"`
	Summary   AnalyticsSummaryStats   `json:"summary"`
}

type AnalyticsSummary struct {
	FromSec             int64   `json:"fromSec"`
	ToSec               int64   `json:"toSec"`
	TotalSessions       int     `json:"totalSessions"`
	TotalCompleted      int     `json:"totalCompleted"`
	TotalSkipped        int     `json:"totalSkipped"`
	TotalCanceled       int     `json:"totalCanceled"`
	CompletionRate      float64 `json:"completionRate"`
	SkipRate            float64 `json:"skipRate"`
	TotalActualBreakSec int     `json:"totalActualBreakSec"`
	AvgActualBreakSec   float64 `json:"avgActualBreakSec"`
}

type AnalyticsTrendPoint struct {
	Day                 string  `json:"day"`
	TotalSessions       int     `json:"totalSessions"`
	TotalCompleted      int     `json:"totalCompleted"`
	TotalSkipped        int     `json:"totalSkipped"`
	TotalCanceled       int     `json:"totalCanceled"`
	CompletionRate      float64 `json:"completionRate"`
	SkipRate            float64 `json:"skipRate"`
	TotalActualBreakSec int     `json:"totalActualBreakSec"`
	AvgActualBreakSec   float64 `json:"avgActualBreakSec"`
}

type AnalyticsTrend struct {
	FromSec int64                 `json:"fromSec"`
	ToSec   int64                 `json:"toSec"`
	Points  []AnalyticsTrendPoint `json:"points"`
}

type AnalyticsBreakTypeDistributionItem struct {
	ReminderID      int64   `json:"reminderId"`
	ReminderName    string  `json:"reminderName"`
	TriggeredCount  int     `json:"triggeredCount"`
	CompletedCount  int     `json:"completedCount"`
	SkippedCount    int     `json:"skippedCount"`
	CanceledCount   int     `json:"canceledCount"`
	CompletionRate  float64 `json:"completionRate"`
	SkipRate        float64 `json:"skipRate"`
	TriggeredShare  float64 `json:"triggeredShare"`
	ReminderType    string  `json:"reminderType,omitempty"`
	ReminderEnabled bool    `json:"reminderEnabled"`
}

type AnalyticsBreakTypeDistribution struct {
	FromSec        int64                                `json:"fromSec"`
	ToSec          int64                                `json:"toSec"`
	TotalTriggered int                                  `json:"totalTriggered"`
	Items          []AnalyticsBreakTypeDistributionItem `json:"items"`
}
