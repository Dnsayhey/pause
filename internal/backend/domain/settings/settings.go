package settings

const (
	TimerModeIdlePause = "idle_pause"
	TimerModeRealTime  = "real_time"
)

type EnforcementSettings struct {
	OverlaySkipAllowed bool `json:"overlaySkipAllowed"`
}

type SoundSettings struct {
	Enabled bool `json:"enabled"`
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

func DefaultSettings() Settings {
	return Settings{
		Enforcement: EnforcementSettings{OverlaySkipAllowed: true},
		Sound:       SoundSettings{Enabled: true},
		Timer: TimerSettings{
			Mode:                  TimerModeIdlePause,
			IdlePauseThresholdSec: 60,
		},
		UI: UISettings{
			ShowTrayCountdown: true,
			Language:          UILanguageAuto,
			Theme:             UIThemeAuto,
		},
	}
}

func (s Settings) Normalize() Settings {
	d := DefaultSettings()

	if s.Timer.IdlePauseThresholdSec <= 0 {
		s.Timer.IdlePauseThresholdSec = d.Timer.IdlePauseThresholdSec
	}
	if s.Timer.Mode != TimerModeIdlePause && s.Timer.Mode != TimerModeRealTime {
		s.Timer.Mode = d.Timer.Mode
	}
	s.UI.Language = NormalizeUILanguage(s.UI.Language)
	s.UI.Theme = NormalizeUITheme(s.UI.Theme)
	if s.UI.Theme == "" {
		s.UI.Theme = d.UI.Theme
	}

	return s
}

func (s Settings) ApplyPatch(p SettingsPatch) Settings {
	if p.Enforcement != nil {
		if p.Enforcement.OverlaySkipAllowed != nil {
			s.Enforcement.OverlaySkipAllowed = *p.Enforcement.OverlaySkipAllowed
		}
	}
	if p.Sound != nil {
		if p.Sound.Enabled != nil {
			s.Sound.Enabled = *p.Sound.Enabled
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
		if p.UI.Theme != nil {
			s.UI.Theme = *p.UI.Theme
		}
	}
	return s.Normalize()
}
