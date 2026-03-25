package reminder

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
