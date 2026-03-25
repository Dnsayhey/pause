package reminder

type Reminder struct {
	ID           int64
	Name         string
	Enabled      bool
	IntervalSec  int
	BreakSec     int
	ReminderType string
}

type Patch struct {
	ID           int64
	Name         *string
	Enabled      *bool
	IntervalSec  *int
	BreakSec     *int
	ReminderType *string
}

type CreateInput struct {
	Name         string
	IntervalSec  int
	BreakSec     int
	Enabled      *bool
	ReminderType *string
}
