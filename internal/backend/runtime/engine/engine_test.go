package engine

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"pause/internal/backend/domain/reminder"
	"pause/internal/backend/runtime/state"
	"pause/internal/backend/storage/settingsjson"
)

type fakeIdleProvider struct{ idleSec int }

func (f *fakeIdleProvider) CurrentIdleSeconds() int { return f.idleSec }

type fakeLockProvider struct{ locked bool }

func (f *fakeLockProvider) IsScreenLocked() bool { return f.locked }

type historyRecorderStub struct{ calls int }

func (s *historyRecorderStub) RecordBreak(_ context.Context, _ time.Time, _ time.Time, _ string, _ int, _ int, _ bool, _ []int64) error {
	s.calls++
	return nil
}

func testEngine(t *testing.T) *Engine {
	t.Helper()
	store, err := settingsjson.OpenStore(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("OpenStore() err=%v", err)
	}
	eng := NewEngine(store, &fakeIdleProvider{}, &fakeLockProvider{}, nil, &historyRecorderStub{})
	eng.SetReminderConfigs([]reminder.Reminder{
		{ID: 1, Name: "Eye", Enabled: true, IntervalSec: 2, BreakSec: 20, ReminderType: "rest"},
		{ID: 2, Name: "Stand", Enabled: true, IntervalSec: 3600, BreakSec: 300, ReminderType: "rest"},
	})
	return eng
}

func reminderByID(t *testing.T, rs []state.ReminderRuntime, id int64) state.ReminderRuntime {
	t.Helper()
	for _, r := range rs {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("missing reminder id=%d", id)
	return state.ReminderRuntime{}
}

func TestEngine_StartBreakNowAndSkip(t *testing.T) {
	eng := testEngine(t)
	now := time.Unix(1_700_000_000, 0)

	rs, err := eng.StartBreakNow(now)
	if err != nil {
		t.Fatalf("StartBreakNow() err=%v", err)
	}
	if rs.CurrentSession == nil {
		t.Fatalf("expected active session")
	}

	rs, err = eng.SkipCurrentBreak(now.Add(time.Second), SkipModeNormal)
	if err != nil {
		t.Fatalf("SkipCurrentBreak() err=%v", err)
	}
	if rs.CurrentSession != nil {
		t.Fatalf("expected no session after skip")
	}
}

func TestEngine_StartBreakNowRejectedWhenGlobalDisabled(t *testing.T) {
	eng := testEngine(t)
	now := time.Unix(1_700_000_000, 0)
	if _, err := eng.Pause(now); err != nil {
		t.Fatalf("Pause() err=%v", err)
	}
	if _, err := eng.StartBreakNow(now.Add(time.Second)); err == nil {
		t.Fatalf("expected StartBreakNow() fail when global disabled")
	}
}

func TestEngine_PauseAndResumeReminder(t *testing.T) {
	eng := testEngine(t)
	now := time.Unix(1_700_000_000, 0)

	rs, err := eng.PauseReminder(1, now)
	if err != nil {
		t.Fatalf("PauseReminder() err=%v", err)
	}
	if !reminderByID(t, rs.Reminders, 1).Paused {
		t.Fatalf("expected reminder paused")
	}

	rs, err = eng.ResumeReminder(1, now.Add(time.Second))
	if err != nil {
		t.Fatalf("ResumeReminder() err=%v", err)
	}
	if reminderByID(t, rs.Reminders, 1).Paused {
		t.Fatalf("expected reminder resumed")
	}
}

func TestEngine_SkipCurrentBreakRejectsInvalidMode(t *testing.T) {
	eng := testEngine(t)
	now := time.Unix(1_700_000_000, 0)
	if _, err := eng.StartBreakNow(now); err != nil {
		t.Fatalf("StartBreakNow() err=%v", err)
	}
	if _, err := eng.SkipCurrentBreak(now.Add(time.Second), SkipMode("bad")); err == nil {
		t.Fatalf("expected invalid mode error")
	}
}
