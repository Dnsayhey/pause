package scheduler

import (
	"testing"

	"pause/internal/core/reminder"
)

const (
	testReminderIDEye   int64 = 1
	testReminderIDStand int64 = 2
)

func defaultReminderFixtures() []reminder.ReminderConfig {
	return []reminder.ReminderConfig{
		{ID: testReminderIDEye, Enabled: true, IntervalSec: 20 * 60, BreakSec: 20, ReminderType: "rest"},
		{ID: testReminderIDStand, Enabled: true, IntervalSec: 60 * 60, BreakSec: 5 * 60, ReminderType: "rest"},
	}
}

func TestEyeReminderTriggersAtDefaultInterval(t *testing.T) {
	s := New()
	reminders := defaultReminderFixtures()
	eye, _ := findReminderByID(reminders, testReminderIDEye)

	evt := s.OnActiveSeconds(eye.IntervalSec-1, reminders)
	if evt != nil {
		t.Fatalf("unexpected event before interval")
	}

	evt = s.OnActiveSeconds(1, reminders)
	if evt == nil {
		t.Fatalf("expected eye reminder event")
	}
	if len(evt.Reasons) != 1 || evt.Reasons[0] != ReminderType(testReminderIDEye) {
		t.Fatalf("unexpected reasons: %#v", evt.Reasons)
	}
	if evt.BreakSec != eye.BreakSec {
		t.Fatalf("expected break %d, got %d", eye.BreakSec, evt.BreakSec)
	}
}

func TestMergeConflictWithinWindow(t *testing.T) {
	s := New()
	reminders := []reminder.ReminderConfig{
		{ID: testReminderIDEye, Enabled: true, IntervalSec: 1200, BreakSec: 20},
		{ID: testReminderIDStand, Enabled: true, IntervalSec: 1230, BreakSec: 300},
	}
	stand, _ := findReminderByID(reminders, testReminderIDStand)

	evt := s.OnActiveSeconds(1200, reminders)
	if evt == nil {
		t.Fatalf("expected merged event")
	}
	if len(evt.Reasons) != 2 {
		t.Fatalf("expected merged reasons, got %#v", evt.Reasons)
	}
	if evt.BreakSec != stand.BreakSec {
		t.Fatalf("expected max break %d, got %d", stand.BreakSec, evt.BreakSec)
	}
}

func TestNextCountdown(t *testing.T) {
	s := New()
	reminders := defaultReminderFixtures()
	eye, _ := findReminderByID(reminders, testReminderIDEye)
	stand, _ := findReminderByID(reminders, testReminderIDStand)

	s.OnActiveSeconds(100, reminders)
	if got := s.NextInSec(reminders, testReminderIDEye); got != eye.IntervalSec-100 {
		t.Fatalf("unexpected next eye in sec: %d", got)
	}
	if got := s.NextInSec(reminders, testReminderIDStand); got != stand.IntervalSec-100 {
		t.Fatalf("unexpected next stand in sec: %d", got)
	}
}
