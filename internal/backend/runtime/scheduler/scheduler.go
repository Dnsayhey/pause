package scheduler

import (
	"sort"

	"pause/internal/backend/domain/reminder"
)

const MergeWindowSec = 60

type ReminderType int64

type Event struct {
	Reasons  []ReminderType
	BreakSec int
}

// Scheduler is not thread-safe and must be called under external synchronization.
type Scheduler struct {
	elapsedSec map[int64]int
}

func New() *Scheduler {
	return &Scheduler{elapsedSec: map[int64]int{}}
}

func (s *Scheduler) ResetByID(id int64) {
	delete(s.elapsedSec, id)
}

func (s *Scheduler) PostponeByID(id int64, intervalSec int, delaySec int) {
	if id <= 0 {
		return
	}
	if intervalSec <= 0 {
		delete(s.elapsedSec, id)
		return
	}
	if delaySec <= 0 {
		s.elapsedSec[id] = intervalSec
		return
	}
	s.elapsedSec[id] = intervalSec - delaySec
}

func (s *Scheduler) OnActiveSeconds(activeSec int, reminders []reminder.Reminder) *Event {
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

	dueIDs := map[int64]struct{}{}
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
		if remaining <= MergeWindowSec {
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

func (s *Scheduler) NextInSec(reminders []reminder.Reminder, reminderID int64) int {
	cfg, ok := findReminderByID(reminders, reminderID)
	if !ok || !cfg.Enabled {
		return -1
	}
	remaining := cfg.IntervalSec - s.elapsedSec[cfg.ID]
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (s *Scheduler) NextByID(reminders []reminder.Reminder) map[int64]int {
	next := map[int64]int{}
	for _, reminder := range reminders {
		if !reminder.Enabled {
			next[reminder.ID] = -1
			continue
		}
		remaining := reminder.IntervalSec - s.elapsedSec[reminder.ID]
		if remaining < 0 {
			remaining = 0
		}
		next[reminder.ID] = remaining
	}
	return next
}

func enabledReminders(reminders []reminder.Reminder) []reminder.Reminder {
	result := make([]reminder.Reminder, 0, len(reminders))
	for _, reminder := range reminders {
		if !reminder.Enabled {
			continue
		}
		result = append(result, reminder)
	}
	return result
}

func findReminderByID(reminders []reminder.Reminder, id int64) (reminder.Reminder, bool) {
	for _, reminder := range reminders {
		if reminder.ID == id {
			return reminder, true
		}
	}
	return reminder.Reminder{}, false
}
