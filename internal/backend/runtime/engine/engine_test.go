package engine

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"pause/internal/backend/domain/reminder"
	settingsdomain "pause/internal/backend/domain/settings"
	"pause/internal/backend/ports"
	"pause/internal/backend/runtime/state"
	"pause/internal/backend/storage/settingsjson"
)

type fakeIdleProvider struct{ idleSec int }

func (f *fakeIdleProvider) CurrentIdleSeconds() int { return f.idleSec }

type fakeLockProvider struct{ locked bool }

func (f *fakeLockProvider) IsScreenLocked() bool { return f.locked }

type historyRecorderStub struct{ calls int }

func (s *historyRecorderStub) RecordBreak(_ context.Context, _ ports.BreakRecordInput) error {
	s.calls++
	return nil
}

type blockingNotifierStub struct {
	started chan struct{}
	done    chan struct{}
}

func (s *blockingNotifierStub) ShowReminder(ctx context.Context, _, _ string) error {
	select {
	case s.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	select {
	case s.done <- struct{}{}:
	default:
	}
	return ctx.Err()
}

func testEngine(t *testing.T) *Engine {
	t.Helper()
	store, err := settingsjson.OpenStore(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("OpenStore() err=%v", err)
	}
	eng := NewEngine(store, &fakeIdleProvider{}, &fakeLockProvider{}, nil, nil, &historyRecorderStub{})
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
	_ = eng.Pause(now)
	if _, err := eng.StartBreakNow(now.Add(time.Second)); err == nil {
		t.Fatalf("expected StartBreakNow() fail when global disabled")
	}
}

func TestEngine_GlobalEnabledDetachedFromSettingsStore(t *testing.T) {
	eng := testEngine(t)
	now := time.Unix(1_700_000_000, 0)
	disabled := false
	if _, err := eng.UpdateSettings(settingsdomain.SettingsPatch{
		Sound: &settingsdomain.SoundSettingsPatch{Enabled: &disabled},
	}); err != nil {
		t.Fatalf("UpdateSettings() err=%v", err)
	}

	rs := eng.GetRuntimeState(now)
	if !rs.GlobalEnabled {
		t.Fatalf("expected runtime global enabled to stay true")
	}
	if _, err := eng.StartBreakNow(now.Add(time.Second)); err != nil {
		t.Fatalf("expected StartBreakNow() to still work, err=%v", err)
	}
}

func TestEngine_PauseResumeFreezesAndContinuesSchedulerProgress(t *testing.T) {
	eng := testEngine(t)
	base := time.Unix(1_700_000_000, 0)

	eng.Tick(base) // bootstrap
	eng.Tick(base.Add(time.Second))

	beforePause := eng.GetRuntimeState(base.Add(time.Second))
	if got := reminderByID(t, beforePause.Reminders, 1).NextInSec; got != 1 {
		t.Fatalf("expected nextIn=1 before pause, got=%d", got)
	}

	_ = eng.Pause(base.Add(time.Second))

	eng.Tick(base.Add(100 * time.Second))
	duringPause := eng.GetRuntimeState(base.Add(100 * time.Second))
	if got := reminderByID(t, duringPause.Reminders, 1).NextInSec; got != 1 {
		t.Fatalf("expected nextIn unchanged during pause, got=%d", got)
	}

	eng.Resume(base.Add(100 * time.Second))
	eng.Tick(base.Add(101 * time.Second))
	afterResume := eng.GetRuntimeState(base.Add(101 * time.Second))
	if afterResume.CurrentSession == nil {
		t.Fatalf("expected break to trigger after resume with continued progress")
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

func TestEngine_StopWaitsForNotificationTasks(t *testing.T) {
	store, err := settingsjson.OpenStore(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("OpenStore() err=%v", err)
	}
	notifier := &blockingNotifierStub{
		started: make(chan struct{}, 1),
		done:    make(chan struct{}, 1),
	}
	eng := NewEngine(store, &fakeIdleProvider{}, &fakeLockProvider{}, nil, notifier, &historyRecorderStub{})
	eng.SetReminderConfigs([]reminder.Reminder{{ID: 1, Name: "Hydrate", Enabled: true, IntervalSec: 60, BreakSec: 1, ReminderType: "notify"}})

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eng.Start(runCtx)
	eng.notifyRemindersLocked([]int64{1}, settingsdomain.UILanguageEnUS)

	select {
	case <-notifier.started:
	case <-time.After(time.Second):
		t.Fatalf("expected notification task to start")
	}

	stopped := make(chan struct{})
	go func() {
		eng.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatalf("expected Stop() to return after canceling background tasks")
	}

	select {
	case <-notifier.done:
	case <-time.After(time.Second):
		t.Fatalf("expected notifier to observe cancellation")
	}
}

func TestEngine_StopAfterCanceledStartContext(t *testing.T) {
	store, err := settingsjson.OpenStore(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("OpenStore() err=%v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	eng := NewEngine(store, &fakeIdleProvider{}, &fakeLockProvider{}, nil, &blockingNotifierStub{
		started: make(chan struct{}, 1),
		done:    make(chan struct{}, 1),
	}, &historyRecorderStub{})
	eng.Start(ctx)

	stopped := make(chan struct{})
	go func() {
		eng.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatalf("expected Stop() to return promptly for canceled start context")
	}

	if err := ctx.Err(); !errors.Is(err, context.Canceled) {
		t.Fatalf("ctx err=%v want=%v", err, context.Canceled)
	}
}
