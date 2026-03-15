package scheduler

import (
	"testing"

	"pause/internal/core/config"
)

func TestEyeReminderTriggersAtDefaultInterval(t *testing.T) {
	s := New()
	cfg := config.DefaultSettings()
	eye, _ := cfg.ReminderByID(config.ReminderIDEye)

	evt := s.OnActiveSeconds(eye.IntervalSec-1, cfg)
	if evt != nil {
		t.Fatalf("unexpected event before interval")
	}

	evt = s.OnActiveSeconds(1, cfg)
	if evt == nil {
		t.Fatalf("expected eye reminder event")
	}
	if len(evt.Reasons) != 1 || evt.Reasons[0] != ReminderEye {
		t.Fatalf("unexpected reasons: %#v", evt.Reasons)
	}
	if evt.BreakSec != eye.BreakSec {
		t.Fatalf("expected break %d, got %d", eye.BreakSec, evt.BreakSec)
	}
}

func TestMergeConflictWithinWindow(t *testing.T) {
	s := New()
	cfg := config.DefaultSettings()
	cfg = cfg.ApplyPatch(config.SettingsPatch{
		Reminders: []config.ReminderPatch{
			{ID: config.ReminderIDEye, IntervalSec: intPtr(1200)},
			{ID: config.ReminderIDStand, IntervalSec: intPtr(1230)},
		},
	})
	stand, _ := cfg.ReminderByID(config.ReminderIDStand)

	evt := s.OnActiveSeconds(1200, cfg)
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
	cfg := config.DefaultSettings()
	eye, _ := cfg.ReminderByID(config.ReminderIDEye)
	stand, _ := cfg.ReminderByID(config.ReminderIDStand)

	s.OnActiveSeconds(100, cfg)
	if got := s.NextInSec(cfg, config.ReminderIDEye); got != eye.IntervalSec-100 {
		t.Fatalf("unexpected next eye in sec: %d", got)
	}
	if got := s.NextInSec(cfg, config.ReminderIDStand); got != stand.IntervalSec-100 {
		t.Fatalf("unexpected next stand in sec: %d", got)
	}
}

func intPtr(v int) *int { return &v }
