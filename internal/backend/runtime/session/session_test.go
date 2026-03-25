package session

import (
	"testing"
	"time"

	"pause/internal/backend/runtime/scheduler"
)

func TestManager_CompletesAfterDuration(t *testing.T) {
	m := NewManager()
	base := time.Unix(1_700_000_000, 0)

	m.StartBreak(base, &scheduler.Event{Reasons: []scheduler.ReminderType{1}, BreakSec: 20}, true)
	if !m.IsActive() {
		t.Fatalf("expected active session")
	}

	m.Tick(base.Add(20 * time.Second))
	view := m.CurrentView(base.Add(20 * time.Second))
	if view == nil || view.Status != string(StatusCompleted) {
		t.Fatalf("expected completed session, got=%#v", view)
	}
}

func TestManager_SkipHonorsCanSkip(t *testing.T) {
	m := NewManager()
	base := time.Unix(1_700_000_000, 0)

	m.StartBreak(base, &scheduler.Event{Reasons: []scheduler.ReminderType{2}, BreakSec: 60}, false)
	if err := m.Skip(); err == nil {
		t.Fatalf("expected skip error when disabled")
	}

	m.StartBreak(base, &scheduler.Event{Reasons: []scheduler.ReminderType{2}, BreakSec: 60}, true)
	if err := m.Skip(); err != nil {
		t.Fatalf("expected skip success, err=%v", err)
	}
	m.ClearIfDone()
	if m.CurrentView(base) != nil {
		t.Fatalf("expected session cleared")
	}
}
