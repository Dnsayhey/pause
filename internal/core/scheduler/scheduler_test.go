package scheduler

import (
	"testing"

	"pause/internal/core/config"
)

func TestEyeReminderTriggersAtDefaultInterval(t *testing.T) {
	s := New()
	cfg := config.DefaultSettings()

	evt := s.OnActiveSeconds(cfg.Eye.IntervalSec-1, cfg)
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
	if evt.BreakSec != cfg.Eye.BreakSec {
		t.Fatalf("expected break %d, got %d", cfg.Eye.BreakSec, evt.BreakSec)
	}
}

func TestMergeConflictWithinWindow(t *testing.T) {
	s := New()
	cfg := config.DefaultSettings()

	cfg.Eye.IntervalSec = 1200
	cfg.Stand.IntervalSec = 1230

	evt := s.OnActiveSeconds(1200, cfg)
	if evt == nil {
		t.Fatalf("expected merged event")
	}
	if len(evt.Reasons) != 2 {
		t.Fatalf("expected merged reasons, got %#v", evt.Reasons)
	}
	if evt.BreakSec != cfg.Stand.BreakSec {
		t.Fatalf("expected max break %d, got %d", cfg.Stand.BreakSec, evt.BreakSec)
	}
}

func TestNextCountdown(t *testing.T) {
	s := New()
	cfg := config.DefaultSettings()

	s.OnActiveSeconds(100, cfg)
	if got := s.NextEyeInSec(cfg); got != cfg.Eye.IntervalSec-100 {
		t.Fatalf("unexpected next eye in sec: %d", got)
	}
	if got := s.NextStandInSec(cfg); got != cfg.Stand.IntervalSec-100 {
		t.Fatalf("unexpected next stand in sec: %d", got)
	}
}
