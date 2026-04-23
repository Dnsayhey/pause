package reminder

import "strings"

const (
	ReminderTypeRest   = "rest"
	ReminderTypeNotify = "notify"
)

type Reminder struct {
	ID           int64  `json:"id"`
	Name         string `json:"name,omitempty"`
	Enabled      bool   `json:"enabled"`
	IntervalSec  int    `json:"intervalSec"`
	BreakSec     int    `json:"breakSec"`
	ReminderType string `json:"reminderType,omitempty"`
}

type Patch struct {
	ID           int64   `json:"id"`
	Name         *string `json:"name,omitempty"`
	Enabled      *bool   `json:"enabled,omitempty"`
	IntervalSec  *int    `json:"intervalSec,omitempty"`
	BreakSec     *int    `json:"breakSec,omitempty"`
	ReminderType *string `json:"reminderType,omitempty"`
}

type CreateInput struct {
	Name         string  `json:"name"`
	IntervalSec  int     `json:"intervalSec"`
	BreakSec     int     `json:"breakSec"`
	Enabled      *bool   `json:"enabled,omitempty"`
	ReminderType *string `json:"reminderType,omitempty"`
}

func NormalizeReminderType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case ReminderTypeRest:
		return ReminderTypeRest
	case ReminderTypeNotify:
		return ReminderTypeNotify
	default:
		return ""
	}
}

func IsRestReminderType(value string) bool {
	return NormalizeReminderType(value) != ReminderTypeNotify
}

func ValidateName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ErrNameRequired
	}
	if name != trimmed {
		return ErrNameTrimmed
	}
	return nil
}

func ValidateReminder(rem Reminder) error {
	if rem.ID <= 0 {
		return ErrIDRequired
	}
	if err := ValidateName(rem.Name); err != nil {
		return err
	}
	if rem.IntervalSec <= 0 {
		return ErrIntervalRange
	}
	if rem.BreakSec <= 0 {
		return ErrBreakRange
	}
	if NormalizeReminderType(rem.ReminderType) == "" {
		return ErrTypeInvalid
	}
	return nil
}

func (in CreateInput) Normalize() (CreateInput, error) {
	if err := ValidateName(in.Name); err != nil {
		return CreateInput{}, err
	}
	if in.IntervalSec <= 0 {
		return CreateInput{}, ErrIntervalRange
	}
	if in.BreakSec <= 0 {
		return CreateInput{}, ErrBreakRange
	}
	if in.ReminderType == nil {
		return CreateInput{}, ErrTypeRequired
	}
	reminderType := NormalizeReminderType(*in.ReminderType)
	if reminderType == "" {
		return CreateInput{}, ErrTypeInvalid
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	return CreateInput{
		Name:         strings.TrimSpace(in.Name),
		IntervalSec:  in.IntervalSec,
		BreakSec:     in.BreakSec,
		Enabled:      &enabled,
		ReminderType: &reminderType,
	}, nil
}

func (p Patch) Normalize() (Patch, error) {
	if p.ID <= 0 {
		return Patch{}, ErrIDRequired
	}
	normalized := p
	if p.Name != nil {
		if err := ValidateName(*p.Name); err != nil {
			return Patch{}, err
		}
		trimmed := strings.TrimSpace(*p.Name)
		normalized.Name = &trimmed
	}
	if p.IntervalSec != nil && *p.IntervalSec <= 0 {
		return Patch{}, ErrIntervalRange
	}
	if p.BreakSec != nil && *p.BreakSec <= 0 {
		return Patch{}, ErrBreakRange
	}
	if p.ReminderType != nil {
		reminderType := NormalizeReminderType(*p.ReminderType)
		if reminderType == "" {
			return Patch{}, ErrTypeInvalid
		}
		normalized.ReminderType = &reminderType
	}
	return normalized, nil
}
