package scheduler

import (
	"testing"

	"pause/internal/backend/domain/reminder"
)

func fixtureReminders() []reminder.Reminder {
	return []reminder.Reminder{
		{ID: 1, Enabled: true, IntervalSec: 20 * 60, BreakSec: 20, ReminderType: "rest"},
		{ID: 2, Enabled: true, IntervalSec: 60 * 60, BreakSec: 5 * 60, ReminderType: "rest"},
	}
}

func TestScheduler_TriggersOnIntervalBoundary(t *testing.T) {
	s := New()
	reminders := fixtureReminders()
	eye := reminders[0]

	if evt := s.OnActiveSeconds(eye.IntervalSec-1, reminders); evt != nil {
		t.Fatalf("unexpected event before boundary")
	}
	evt := s.OnActiveSeconds(1, reminders)
	if evt == nil {
		t.Fatalf("expected boundary event")
	}
	if len(evt.Reasons) != 1 || evt.Reasons[0] != ReminderType(1) {
		t.Fatalf("unexpected reasons=%v", evt.Reasons)
	}
	if evt.BreakSec != eye.BreakSec {
		t.Fatalf("break mismatch got=%d want=%d", evt.BreakSec, eye.BreakSec)
	}
}

func TestScheduler_MergesNearbyReminders(t *testing.T) {
	s := New()
	reminders := []reminder.Reminder{
		{ID: 1, Enabled: true, IntervalSec: 1200, BreakSec: 20, ReminderType: "rest"},
		{ID: 2, Enabled: true, IntervalSec: 1230, BreakSec: 300, ReminderType: "rest"},
	}
	if evt := s.OnActiveSeconds(1200, reminders); evt == nil || len(evt.Reasons) != 2 || evt.BreakSec != 300 {
		t.Fatalf("expected merged event, got=%#v", evt)
	}
}

func TestScheduler_NextInSec(t *testing.T) {
	s := New()
	reminders := fixtureReminders()
	s.OnActiveSeconds(100, reminders)

	if got := s.NextInSec(reminders, 1); got != reminders[0].IntervalSec-100 {
		t.Fatalf("next eye mismatch got=%d", got)
	}
	if got := s.NextInSec(reminders, 2); got != reminders[1].IntervalSec-100 {
		t.Fatalf("next stand mismatch got=%d", got)
	}
}
