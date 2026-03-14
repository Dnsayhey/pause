package scheduler

import (
	"sort"

	"pause/internal/core/config"
)

const mergeWindowSec = 60

type ReminderType string

const (
	ReminderEye   ReminderType = "eye"
	ReminderStand ReminderType = "stand"
)

type Event struct {
	Reasons  []ReminderType
	BreakSec int
}

type Scheduler struct {
	eyeElapsedSec   int
	standElapsedSec int
}

func New() *Scheduler {
	return &Scheduler{}
}

func (s *Scheduler) Reset() {
	s.eyeElapsedSec = 0
	s.standElapsedSec = 0
}

func (s *Scheduler) ResetEye() {
	s.eyeElapsedSec = 0
}

func (s *Scheduler) ResetStand() {
	s.standElapsedSec = 0
}

func (s *Scheduler) OnActiveSeconds(activeSec int, settings config.Settings) *Event {
	if activeSec <= 0 {
		return nil
	}

	if settings.Eye.Enabled {
		s.eyeElapsedSec += activeSec
	}
	if settings.Stand.Enabled {
		s.standElapsedSec += activeSec
	}

	eyeDue := settings.Eye.Enabled && s.eyeElapsedSec >= settings.Eye.IntervalSec
	standDue := settings.Stand.Enabled && s.standElapsedSec >= settings.Stand.IntervalSec

	// Merge reminders if they are due in the same minute-sized window to avoid double interruptions.
	if eyeDue && !standDue && settings.Stand.Enabled {
		if settings.Stand.IntervalSec-s.standElapsedSec <= mergeWindowSec {
			standDue = true
		}
	}
	if standDue && !eyeDue && settings.Eye.Enabled {
		if settings.Eye.IntervalSec-s.eyeElapsedSec <= mergeWindowSec {
			eyeDue = true
		}
	}

	if !eyeDue && !standDue {
		return nil
	}

	reasons := make([]ReminderType, 0, 2)
	breakSec := 0

	if eyeDue {
		reasons = append(reasons, ReminderEye)
		s.eyeElapsedSec = 0
		if settings.Eye.BreakSec > breakSec {
			breakSec = settings.Eye.BreakSec
		}
	}
	if standDue {
		reasons = append(reasons, ReminderStand)
		s.standElapsedSec = 0
		if settings.Stand.BreakSec > breakSec {
			breakSec = settings.Stand.BreakSec
		}
	}

	sort.Slice(reasons, func(i, j int) bool { return reasons[i] < reasons[j] })
	return &Event{Reasons: reasons, BreakSec: breakSec}
}

func (s *Scheduler) NextEyeInSec(settings config.Settings) int {
	if !settings.Eye.Enabled {
		return -1
	}
	remaining := settings.Eye.IntervalSec - s.eyeElapsedSec
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (s *Scheduler) NextStandInSec(settings config.Settings) int {
	if !settings.Stand.Enabled {
		return -1
	}
	remaining := settings.Stand.IntervalSec - s.standElapsedSec
	if remaining < 0 {
		return 0
	}
	return remaining
}
