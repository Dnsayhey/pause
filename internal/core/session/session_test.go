package session

import (
	"testing"
	"time"

	"pause/internal/core/scheduler"
)

func TestSessionCompletesAfterDuration(t *testing.T) {
	m := NewManager()
	now := time.Unix(1_700_000_000, 0)

	m.StartBreak(now, &scheduler.Event{Reasons: []scheduler.ReminderType{"eye"}, BreakSec: 20}, true)
	if !m.IsActive() {
		t.Fatalf("expected active session after StartBreak")
	}

	m.Tick(now.Add(20 * time.Second))
	view := m.CurrentView(now.Add(20 * time.Second))
	if view == nil || view.Status != string(StatusCompleted) {
		t.Fatalf("expected completed session, got %#v", view)
	}
}

func TestSkipRespectsCanSkip(t *testing.T) {
	m := NewManager()
	now := time.Unix(1_700_000_000, 0)

	m.StartBreak(now, &scheduler.Event{Reasons: []scheduler.ReminderType{"stand"}, BreakSec: 60}, false)
	if err := m.Skip(); err == nil {
		t.Fatalf("expected error when skip is disabled")
	}

	m.StartBreak(now, &scheduler.Event{Reasons: []scheduler.ReminderType{"stand"}, BreakSec: 60}, true)
	if err := m.Skip(); err != nil {
		t.Fatalf("expected skip success, got %v", err)
	}
	m.ClearIfDone()
	if m.CurrentView(now) != nil {
		t.Fatalf("expected session to be cleared")
	}
}
