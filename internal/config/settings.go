package config

import "time"

const (
	TimerModeIdlePause = "idle_pause"
	TimerModeRealTime  = "real_time"
)

type ReminderRule struct {
	Enabled     bool `json:"enabled"`
	IntervalSec int  `json:"intervalSec"`
	BreakSec    int  `json:"breakSec"`
}

type EnforcementSettings struct {
	OverlayEnabled     bool `json:"overlayEnabled"`
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
}

type StartupSettings struct {
	LaunchAtLogin bool `json:"launchAtLogin"`
}

type Settings struct {
	GlobalEnabled bool                `json:"globalEnabled"`
	Eye           ReminderRule        `json:"eye"`
	Stand         ReminderRule        `json:"stand"`
	Enforcement   EnforcementSettings `json:"enforcement"`
	Sound         SoundSettings       `json:"sound"`
	Timer         TimerSettings       `json:"timer"`
	UI            UISettings          `json:"ui"`
	Startup       StartupSettings     `json:"startup"`
}

type ReminderRulePatch struct {
	Enabled     *bool `json:"enabled,omitempty"`
	IntervalSec *int  `json:"intervalSec,omitempty"`
	BreakSec    *int  `json:"breakSec,omitempty"`
}

type EnforcementSettingsPatch struct {
	OverlayEnabled     *bool `json:"overlayEnabled,omitempty"`
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
}

type StartupSettingsPatch struct {
	LaunchAtLogin *bool `json:"launchAtLogin,omitempty"`
}

type SettingsPatch struct {
	GlobalEnabled *bool                     `json:"globalEnabled,omitempty"`
	Eye           *ReminderRulePatch        `json:"eye,omitempty"`
	Stand         *ReminderRulePatch        `json:"stand,omitempty"`
	Enforcement   *EnforcementSettingsPatch `json:"enforcement,omitempty"`
	Sound         *SoundSettingsPatch       `json:"sound,omitempty"`
	Timer         *TimerSettingsPatch       `json:"timer,omitempty"`
	UI            *UISettingsPatch          `json:"ui,omitempty"`
	Startup       *StartupSettingsPatch     `json:"startup,omitempty"`
}

type RuntimeState struct {
	Now                time.Time         `json:"now"`
	Paused             bool              `json:"paused"`
	PauseMode          string            `json:"pauseMode,omitempty"`
	PausedUntil        *time.Time        `json:"pausedUntil,omitempty"`
	CurrentSession     *BreakSessionView `json:"currentSession,omitempty"`
	NextEyeInSec       int               `json:"nextEyeInSec"`
	NextStandInSec     int               `json:"nextStandInSec"`
	NextBreakReason    []string          `json:"nextBreakReason"`
	GlobalEnabled      bool              `json:"globalEnabled"`
	TimerMode          string            `json:"timerMode"`
	IdleThresholdSec   int               `json:"idleThresholdSec"`
	LastTickActive     bool              `json:"lastTickActive"`
	CurrentIdleSec     int               `json:"currentIdleSec"`
	ShowTrayCountdown  bool              `json:"showTrayCountdown"`
	OverlayEnabled     bool              `json:"overlayEnabled"`
	OverlaySkipAllowed bool              `json:"overlaySkipAllowed"`
	OverlayNative      bool              `json:"overlayNative"`
}

type BreakSessionView struct {
	Status       string    `json:"status"`
	Reasons      []string  `json:"reasons"`
	StartedAt    time.Time `json:"startedAt"`
	EndsAt       time.Time `json:"endsAt"`
	RemainingSec int       `json:"remainingSec"`
	CanSkip      bool      `json:"canSkip"`
}

func DefaultSettings() Settings {
	return Settings{
		GlobalEnabled: true,
		Eye: ReminderRule{
			Enabled:     true,
			IntervalSec: 20 * 60,
			BreakSec:    20,
		},
		Stand: ReminderRule{
			Enabled:     true,
			IntervalSec: 60 * 60,
			BreakSec:    5 * 60,
		},
		Enforcement: EnforcementSettings{
			OverlayEnabled:     true,
			OverlaySkipAllowed: true,
		},
		Sound: SoundSettings{
			Enabled: true,
			Volume:  70,
		},
		Timer: TimerSettings{
			Mode:                  TimerModeIdlePause,
			IdlePauseThresholdSec: 300,
		},
		UI: UISettings{
			ShowTrayCountdown: true,
			Language:          UILanguageAuto,
		},
		Startup: StartupSettings{
			LaunchAtLogin: false,
		},
	}
}

func (s Settings) Normalize() Settings {
	d := DefaultSettings()

	if s.Eye.IntervalSec <= 0 {
		s.Eye.IntervalSec = d.Eye.IntervalSec
	}
	if s.Eye.BreakSec <= 0 {
		s.Eye.BreakSec = d.Eye.BreakSec
	}
	if s.Stand.IntervalSec <= 0 {
		s.Stand.IntervalSec = d.Stand.IntervalSec
	}
	if s.Stand.BreakSec <= 0 {
		s.Stand.BreakSec = d.Stand.BreakSec
	}
	if s.Sound.Volume <= 0 || s.Sound.Volume > 100 {
		s.Sound.Volume = d.Sound.Volume
	}
	if s.Timer.IdlePauseThresholdSec <= 0 {
		s.Timer.IdlePauseThresholdSec = d.Timer.IdlePauseThresholdSec
	}
	if s.Timer.Mode != TimerModeIdlePause && s.Timer.Mode != TimerModeRealTime {
		s.Timer.Mode = d.Timer.Mode
	}
	s.UI.Language = NormalizeUILanguage(s.UI.Language)
	if s.UI.Language == "" {
		s.UI.Language = d.UI.Language
	}

	return s
}

func (s Settings) ApplyPatch(p SettingsPatch) Settings {
	if p.GlobalEnabled != nil {
		s.GlobalEnabled = *p.GlobalEnabled
	}
	if p.Eye != nil {
		if p.Eye.Enabled != nil {
			s.Eye.Enabled = *p.Eye.Enabled
		}
		if p.Eye.IntervalSec != nil {
			s.Eye.IntervalSec = *p.Eye.IntervalSec
		}
		if p.Eye.BreakSec != nil {
			s.Eye.BreakSec = *p.Eye.BreakSec
		}
	}
	if p.Stand != nil {
		if p.Stand.Enabled != nil {
			s.Stand.Enabled = *p.Stand.Enabled
		}
		if p.Stand.IntervalSec != nil {
			s.Stand.IntervalSec = *p.Stand.IntervalSec
		}
		if p.Stand.BreakSec != nil {
			s.Stand.BreakSec = *p.Stand.BreakSec
		}
	}
	if p.Enforcement != nil {
		if p.Enforcement.OverlayEnabled != nil {
			s.Enforcement.OverlayEnabled = *p.Enforcement.OverlayEnabled
		}
		if p.Enforcement.OverlaySkipAllowed != nil {
			s.Enforcement.OverlaySkipAllowed = *p.Enforcement.OverlaySkipAllowed
		}
	}
	if p.Sound != nil {
		if p.Sound.Enabled != nil {
			s.Sound.Enabled = *p.Sound.Enabled
		}
		if p.Sound.Volume != nil {
			s.Sound.Volume = *p.Sound.Volume
		}
	}
	if p.Timer != nil {
		if p.Timer.Mode != nil {
			s.Timer.Mode = *p.Timer.Mode
		}
		if p.Timer.IdlePauseThresholdSec != nil {
			s.Timer.IdlePauseThresholdSec = *p.Timer.IdlePauseThresholdSec
		}
	}
	if p.UI != nil {
		if p.UI.ShowTrayCountdown != nil {
			s.UI.ShowTrayCountdown = *p.UI.ShowTrayCountdown
		}
		if p.UI.Language != nil {
			s.UI.Language = *p.UI.Language
		}
	}
	if p.Startup != nil {
		if p.Startup.LaunchAtLogin != nil {
			s.Startup.LaunchAtLogin = *p.Startup.LaunchAtLogin
		}
	}

	return s.Normalize()
}
