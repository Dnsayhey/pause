package scheduler

import (
	"sort"

	"pause/internal/core/config"
)

const mergeWindowSec = 60

type ReminderType string

const (
	ReminderEye   ReminderType = ReminderType(config.ReminderIDEye)
	ReminderStand ReminderType = ReminderType(config.ReminderIDStand)
)

type Event struct {
	Reasons  []ReminderType
	BreakSec int
}

type Scheduler struct {
	elapsedSec map[string]int
}

func New() *Scheduler {
	return &Scheduler{elapsedSec: map[string]int{}}
}

func (s *Scheduler) Reset() {
	s.elapsedSec = map[string]int{}
}

func (s *Scheduler) ResetByID(id string) {
	norm := config.NormalizeReminderID(id)
	if norm == "" {
		return
	}
	delete(s.elapsedSec, norm)
}

func (s *Scheduler) OnActiveSeconds(activeSec int, reminders []config.ReminderConfig) *Event {
	if activeSec <= 0 {
		return nil
	}

	enabled := enabledReminders(reminders)
	if len(enabled) == 0 {
		return nil
	}

	for _, reminder := range enabled {
		s.elapsedSec[reminder.ID] += activeSec
	}

	dueIDs := map[string]struct{}{}
	for _, reminder := range enabled {
		if s.elapsedSec[reminder.ID] >= reminder.IntervalSec {
			dueIDs[reminder.ID] = struct{}{}
		}
	}
	if len(dueIDs) == 0 {
		return nil
	}

	// Merge reminders that are close enough to avoid back-to-back interruptions.
	for _, reminder := range enabled {
		if _, alreadyDue := dueIDs[reminder.ID]; alreadyDue {
			continue
		}
		remaining := reminder.IntervalSec - s.elapsedSec[reminder.ID]
		if remaining <= mergeWindowSec {
			dueIDs[reminder.ID] = struct{}{}
		}
	}

	reasons := make([]ReminderType, 0, len(dueIDs))
	breakSec := 0
	for _, reminder := range enabled {
		if _, ok := dueIDs[reminder.ID]; !ok {
			continue
		}
		reasons = append(reasons, ReminderType(reminder.ID))
		s.elapsedSec[reminder.ID] = 0
		if reminder.BreakSec > breakSec {
			breakSec = reminder.BreakSec
		}
	}
	if len(reasons) == 0 {
		return nil
	}

	sort.Slice(reasons, func(i, j int) bool { return reasons[i] < reasons[j] })
	return &Event{Reasons: reasons, BreakSec: breakSec}
}

func (s *Scheduler) NextInSec(reminders []config.ReminderConfig, reminderID string) int {
	reminder, ok := config.ReminderByID(reminders, reminderID)
	if !ok || !reminder.Enabled {
		return -1
	}
	remaining := reminder.IntervalSec - s.elapsedSec[reminder.ID]
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (s *Scheduler) NextByID(reminders []config.ReminderConfig) map[string]int {
	next := map[string]int{}
	for _, reminder := range reminders {
		next[reminder.ID] = s.NextInSec(reminders, reminder.ID)
	}
	return next
}

func enabledReminders(reminders []config.ReminderConfig) []config.ReminderConfig {
	result := make([]config.ReminderConfig, 0, len(reminders))
	for _, reminder := range reminders {
		if !reminder.Enabled {
			continue
		}
		result = append(result, reminder)
	}
	return result
}
