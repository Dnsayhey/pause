package config

import (
	"strings"
	"time"
)

const (
	TimerModeIdlePause = "idle_pause"
	TimerModeRealTime  = "real_time"
)

const (
	ReminderIDEye   = "eye"
	ReminderIDStand = "stand"
)

type ReminderConfig struct {
	ID           string `json:"id"`
	Name         string `json:"name,omitempty"`
	Enabled      bool   `json:"enabled"`
	IntervalSec  int    `json:"intervalSec"`
	BreakSec     int    `json:"breakSec"`
	DeliveryType string `json:"deliveryType,omitempty"`
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
	GlobalEnabled bool                `json:"globalEnabled"`
	Reminders     []ReminderConfig    `json:"reminders,omitempty"`
	Enforcement   EnforcementSettings `json:"enforcement"`
	Sound         SoundSettings       `json:"sound"`
	Timer         TimerSettings       `json:"timer"`
	UI            UISettings          `json:"ui"`
}

type ReminderPatch struct {
	ID           string  `json:"id"`
	Name         *string `json:"name,omitempty"`
	Enabled      *bool   `json:"enabled,omitempty"`
	IntervalSec  *int    `json:"intervalSec,omitempty"`
	BreakSec     *int    `json:"breakSec,omitempty"`
	DeliveryType *string `json:"deliveryType,omitempty"`
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
	GlobalEnabled *bool                     `json:"globalEnabled,omitempty"`
	Reminders     []ReminderPatch           `json:"reminders,omitempty"`
	Enforcement   *EnforcementSettingsPatch `json:"enforcement,omitempty"`
	Sound         *SoundSettingsPatch       `json:"sound,omitempty"`
	Timer         *TimerSettingsPatch       `json:"timer,omitempty"`
	UI            *UISettingsPatch          `json:"ui,omitempty"`
}

type ReminderRuntime struct {
	ID          string `json:"id"`
	Enabled     bool   `json:"enabled"`
	Paused      bool   `json:"paused"`
	NextInSec   int    `json:"nextInSec"`
	IntervalSec int    `json:"intervalSec"`
	BreakSec    int    `json:"breakSec"`
}

type RuntimeState struct {
	Now                time.Time         `json:"now"`
	CurrentSession     *BreakSessionView `json:"currentSession,omitempty"`
	Reminders          []ReminderRuntime `json:"reminders"`
	NextBreakReason    []string          `json:"nextBreakReason"`
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
	Reasons      []string  `json:"reasons"`
	StartedAt    time.Time `json:"startedAt"`
	EndsAt       time.Time `json:"endsAt"`
	RemainingSec int       `json:"remainingSec"`
	CanSkip      bool      `json:"canSkip"`
}

func DefaultSettings() Settings {
	return Settings{
		GlobalEnabled: true,
		Reminders: []ReminderConfig{
			{ID: ReminderIDEye, Enabled: true, IntervalSec: 20 * 60, BreakSec: 20},
			{ID: ReminderIDStand, Enabled: true, IntervalSec: 60 * 60, BreakSec: 5 * 60},
		},
		Enforcement: EnforcementSettings{OverlaySkipAllowed: true},
		Sound:       SoundSettings{Enabled: true, Volume: 70},
		Timer: TimerSettings{
			Mode:                  TimerModeIdlePause,
			IdlePauseThresholdSec: 300,
		},
		UI: UISettings{
			ShowTrayCountdown: true,
			Language:          UILanguageAuto,
			Theme:             UIThemeAuto,
		},
	}
}

func NormalizeReminderID(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}

func (s Settings) Normalize() Settings {
	d := DefaultSettings()

	s.Reminders = normalizeReminders(s.Reminders)
	if len(s.Reminders) == 0 {
		s.Reminders = d.Reminders
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
	s.UI.Theme = NormalizeUITheme(s.UI.Theme)
	if s.UI.Theme == "" {
		s.UI.Theme = d.UI.Theme
	}

	return s
}

func normalizeReminders(reminders []ReminderConfig) []ReminderConfig {
	if len(reminders) == 0 {
		return nil
	}

	result := make([]ReminderConfig, 0, len(reminders))
	indexByID := map[string]int{}
	for _, reminder := range reminders {
		id := NormalizeReminderID(reminder.ID)
		if id == "" {
			continue
		}
		intervalDef, breakDef := reminderDefaultsForID(id)
		next := ReminderConfig{
			ID:           id,
			Name:         strings.TrimSpace(reminder.Name),
			Enabled:      reminder.Enabled,
			IntervalSec:  reminder.IntervalSec,
			BreakSec:     reminder.BreakSec,
			DeliveryType: normalizeReminderDeliveryType(reminder.DeliveryType),
		}
		if next.IntervalSec <= 0 {
			next.IntervalSec = intervalDef
		}
		if next.BreakSec <= 0 {
			next.BreakSec = breakDef
		}

		if idx, ok := indexByID[id]; ok {
			result[idx] = next
			continue
		}
		indexByID[id] = len(result)
		result = append(result, next)
	}
	return result
}

func reminderDefaultsForID(id string) (intervalSec int, breakSec int) {
	switch NormalizeReminderID(id) {
	case ReminderIDEye:
		return 20 * 60, 20
	case ReminderIDStand:
		return 60 * 60, 5 * 60
	default:
		return 20 * 60, 20
	}
}

func (s Settings) ReminderByID(id string) (ReminderConfig, bool) {
	norm := NormalizeReminderID(id)
	for _, reminder := range s.Reminders {
		if reminder.ID == norm {
			return reminder, true
		}
	}
	return ReminderConfig{}, false
}

func (s Settings) ApplyPatch(p SettingsPatch) Settings {
	s.Reminders = append([]ReminderConfig(nil), s.Reminders...)
	if p.GlobalEnabled != nil {
		s.GlobalEnabled = *p.GlobalEnabled
	}
	if len(p.Reminders) > 0 {
		for _, reminderPatch := range p.Reminders {
			s.Reminders = applyReminderPatch(s.Reminders, reminderPatch.ID, reminderPatchMutation{
				Name:         reminderPatch.Name,
				Enabled:      reminderPatch.Enabled,
				IntervalSec:  reminderPatch.IntervalSec,
				BreakSec:     reminderPatch.BreakSec,
				DeliveryType: reminderPatch.DeliveryType,
			})
		}
	}
	if p.Enforcement != nil {
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
		if p.UI.Theme != nil {
			s.UI.Theme = *p.UI.Theme
		}
	}
	return s.Normalize()
}

type reminderPatchMutation struct {
	Name         *string
	Enabled      *bool
	IntervalSec  *int
	BreakSec     *int
	DeliveryType *string
}

func applyReminderPatch(reminders []ReminderConfig, reminderID string, patch reminderPatchMutation) []ReminderConfig {
	id := NormalizeReminderID(reminderID)
	if id == "" {
		return reminders
	}

	idx := -1
	for i, reminder := range reminders {
		if reminder.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		intervalDef, breakDef := reminderDefaultsForID(id)
		reminders = append(reminders, ReminderConfig{ID: id, Enabled: true, IntervalSec: intervalDef, BreakSec: breakDef})
		idx = len(reminders) - 1
	}

	if patch.Name != nil {
		name := strings.TrimSpace(*patch.Name)
		if name != "" {
			reminders[idx].Name = name
		}
	}
	if patch.Enabled != nil {
		reminders[idx].Enabled = *patch.Enabled
	}
	if patch.IntervalSec != nil {
		reminders[idx].IntervalSec = *patch.IntervalSec
	}
	if patch.BreakSec != nil {
		reminders[idx].BreakSec = *patch.BreakSec
	}
	if patch.DeliveryType != nil {
		deliveryType := normalizeReminderDeliveryType(*patch.DeliveryType)
		if deliveryType != "" {
			reminders[idx].DeliveryType = deliveryType
		}
	}
	return reminders
}

func normalizeReminderDeliveryType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "overlay":
		return "overlay"
	case "notification":
		return "notification"
	default:
		return ""
	}
}
