package state

import "time"

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

type BreakSessionView struct {
	Status       string    `json:"status"`
	Reasons      []int64   `json:"reasons"`
	StartedAt    time.Time `json:"startedAt"`
	EndsAt       time.Time `json:"endsAt"`
	RemainingSec int       `json:"remainingSec"`
	CanSkip      bool      `json:"canSkip"`
	CanPostpone  bool      `json:"canPostpone"`
}
