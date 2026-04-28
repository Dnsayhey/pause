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

type blockingHistoryRecorderStub struct {
	started chan struct{}
	block   chan struct{}
}

func (s *blockingHistoryRecorderStub) RecordBreak(_ context.Context, _ ports.BreakRecordInput) error {
	select {
	case s.started <- struct{}{}:
	default:
	}
	<-s.block
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

type controllableSettingsStore struct {
	current   settingsdomain.Settings
	updateCh  chan struct{}
	releaseCh chan struct{}
}

func (s *controllableSettingsStore) Get() settingsdomain.Settings {
	return s.current
}

func (s *controllableSettingsStore) Update(patch settingsdomain.SettingsPatch) (settingsdomain.Settings, error) {
	if s.updateCh != nil {
		select {
		case s.updateCh <- struct{}{}:
		default:
		}
	}
	if s.releaseCh != nil {
		<-s.releaseCh
	}
	s.current = s.current.ApplyPatch(patch)
	return s.current, nil
}

type countingNotifierStub struct {
	started chan struct{}
	block   chan struct{}
}

func (s *countingNotifierStub) ShowReminder(ctx context.Context, _, _ string) error {
	select {
	case s.started <- struct{}{}:
	default:
	}
	select {
	case <-s.block:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
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

func TestEngine_SkipCurrentBreakDoesNotHoldLockDuringHistoryWrite(t *testing.T) {
	store, err := settingsjson.OpenStore(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("OpenStore() err=%v", err)
	}
	history := &blockingHistoryRecorderStub{
		started: make(chan struct{}, 1),
		block:   make(chan struct{}),
	}
	eng := NewEngine(store, &fakeIdleProvider{}, &fakeLockProvider{}, nil, nil, history)
	eng.SetReminderConfigs([]reminder.Reminder{
		{ID: 1, Name: "Eye", Enabled: true, IntervalSec: 2, BreakSec: 20, ReminderType: "rest"},
	})

	now := time.Unix(1_700_000_000, 0)
	if _, err := eng.StartBreakNow(now); err != nil {
		t.Fatalf("StartBreakNow() err=%v", err)
	}

	skipDone := make(chan struct{})
	go func() {
		if _, err := eng.SkipCurrentBreak(now.Add(time.Second), SkipModeNormal); err != nil {
			t.Errorf("SkipCurrentBreak() err=%v", err)
		}
		close(skipDone)
	}()

	select {
	case <-history.started:
	case <-time.After(time.Second):
		t.Fatalf("expected history write to start")
	}

	readDone := make(chan struct{})
	go func() {
		_ = eng.GetRuntimeState(now.Add(2 * time.Second))
		close(readDone)
	}()

	select {
	case <-readDone:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected runtime read to proceed while history write is blocked")
	}

	close(history.block)

	select {
	case <-skipDone:
	case <-time.After(time.Second):
		t.Fatalf("expected SkipCurrentBreak() to finish after history write release")
	}
}

func TestEngine_TickCommitsCompletedHistoryBeforeIdleEarlyReturn(t *testing.T) {
	store, err := settingsjson.OpenStore(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("OpenStore() err=%v", err)
	}
	idleProvider := &fakeIdleProvider{}
	history := &historyRecorderStub{}
	eng := NewEngine(store, idleProvider, &fakeLockProvider{}, nil, nil, history)
	eng.SetReminderConfigs([]reminder.Reminder{
		{ID: 1, Name: "Eye", Enabled: true, IntervalSec: 2, BreakSec: 20, ReminderType: "rest"},
	})

	base := time.Unix(1_700_000_000, 0)
	if _, err := eng.StartBreakNow(base); err != nil {
		t.Fatalf("StartBreakNow() err=%v", err)
	}

	idleProvider.idleSec = settingsdomain.DefaultSettings().Timer.IdlePauseThresholdSec + 1
	eng.Tick(base.Add(20 * time.Second))

	if history.calls != 1 {
		t.Fatalf("history writes=%d want=1", history.calls)
	}
}

func TestEngine_PostponeCurrentBreakDelaysWithoutHistory(t *testing.T) {
	store, err := settingsjson.OpenStore(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("OpenStore() err=%v", err)
	}
	history := &historyRecorderStub{}
	eng := NewEngine(store, &fakeIdleProvider{}, &fakeLockProvider{}, nil, nil, history)
	eng.SetReminderConfigs([]reminder.Reminder{
		{ID: 1, Name: "Stand", Enabled: true, IntervalSec: 120, BreakSec: 90, ReminderType: "rest"},
	})

	base := time.Unix(1_700_000_000, 0)
	eng.Tick(base)
	eng.Tick(base.Add(120 * time.Second))
	active := eng.GetRuntimeState(base.Add(120 * time.Second))
	if active.CurrentSession == nil || !active.CurrentSession.CanPostpone {
		t.Fatalf("expected postponable session, got=%#v", active.CurrentSession)
	}

	rs, err := eng.PostponeCurrentBreak(base.Add(121 * time.Second))
	if err != nil {
		t.Fatalf("PostponeCurrentBreak() err=%v", err)
	}
	if rs.CurrentSession != nil {
		t.Fatalf("expected no session after postpone")
	}
	if history.calls != 0 {
		t.Fatalf("history writes=%d want=0", history.calls)
	}
	if got := reminderByID(t, rs.Reminders, 1).NextInSec; got != 60 {
		t.Fatalf("postponed nextIn=%d want=60", got)
	}

	eng.Tick(base.Add(181 * time.Second))
	reopened := eng.GetRuntimeState(base.Add(181 * time.Second))
	if reopened.CurrentSession == nil {
		t.Fatalf("expected session after postponed delay")
	}
	if reopened.CurrentSession.CanPostpone {
		t.Fatalf("expected repostponed session to disallow postpone")
	}
	if _, err := eng.PostponeCurrentBreak(base.Add(182 * time.Second)); err == nil {
		t.Fatalf("expected second postpone to fail")
	}
}

func TestEngine_PostponeCurrentBreakRejectsShortBreak(t *testing.T) {
	eng := testEngine(t)
	base := time.Unix(1_700_000_000, 0)
	eng.SetReminderConfigs([]reminder.Reminder{
		{ID: 1, Name: "Eye", Enabled: true, IntervalSec: 120, BreakSec: 60, ReminderType: "rest"},
	})

	eng.Tick(base)
	eng.Tick(base.Add(120 * time.Second))
	rs := eng.GetRuntimeState(base.Add(120 * time.Second))
	if rs.CurrentSession == nil {
		t.Fatalf("expected active session")
	}
	if rs.CurrentSession.CanPostpone {
		t.Fatalf("expected short break to disallow postpone")
	}
	if _, err := eng.PostponeCurrentBreak(base.Add(121 * time.Second)); err == nil {
		t.Fatalf("expected postpone to fail for short break")
	}
}

func TestEngine_PostponeCurrentBreakHandlesMergedReasons(t *testing.T) {
	store, err := settingsjson.OpenStore(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("OpenStore() err=%v", err)
	}
	eng := NewEngine(store, &fakeIdleProvider{}, &fakeLockProvider{}, nil, nil, &historyRecorderStub{})
	eng.SetReminderConfigs([]reminder.Reminder{
		{ID: 1, Name: "Stand", Enabled: true, IntervalSec: 120, BreakSec: 90, ReminderType: "rest"},
		{ID: 2, Name: "Stretch", Enabled: true, IntervalSec: 150, BreakSec: 120, ReminderType: "rest"},
	})

	base := time.Unix(1_700_000_000, 0)
	eng.Tick(base)
	eng.Tick(base.Add(120 * time.Second))
	rs, err := eng.PostponeCurrentBreak(base.Add(121 * time.Second))
	if err != nil {
		t.Fatalf("PostponeCurrentBreak() err=%v", err)
	}
	if got := reminderByID(t, rs.Reminders, 1).NextInSec; got != 60 {
		t.Fatalf("postponed reason 1 nextIn=%d want=60", got)
	}
	if got := reminderByID(t, rs.Reminders, 2).NextInSec; got != 60 {
		t.Fatalf("postponed reason 2 nextIn=%d want=60", got)
	}
}

func TestEngine_PostponeCurrentBreakResetsAfterCompletion(t *testing.T) {
	store, err := settingsjson.OpenStore(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("OpenStore() err=%v", err)
	}
	eng := NewEngine(store, &fakeIdleProvider{}, &fakeLockProvider{}, nil, nil, &historyRecorderStub{})
	eng.SetReminderConfigs([]reminder.Reminder{
		{ID: 1, Name: "Stand", Enabled: true, IntervalSec: 120, BreakSec: 90, ReminderType: "rest"},
	})

	base := time.Unix(1_700_000_000, 0)
	eng.Tick(base)
	eng.Tick(base.Add(120 * time.Second))
	if _, err := eng.PostponeCurrentBreak(base.Add(121 * time.Second)); err != nil {
		t.Fatalf("PostponeCurrentBreak() err=%v", err)
	}
	eng.Tick(base.Add(181 * time.Second))
	eng.Tick(base.Add(271 * time.Second))
	completed := eng.GetRuntimeState(base.Add(271 * time.Second))
	if completed.CurrentSession != nil {
		t.Fatalf("expected postponed session completed")
	}

	eng.Tick(base.Add(391 * time.Second))
	next := eng.GetRuntimeState(base.Add(391 * time.Second))
	if next.CurrentSession == nil || !next.CurrentSession.CanPostpone {
		t.Fatalf("expected next cycle to allow postpone, got=%#v", next.CurrentSession)
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

func TestEngine_UpdateSettingsDoesNotBlockRuntimeReadsOnStoreIO(t *testing.T) {
	store := &controllableSettingsStore{
		current:   settingsdomain.DefaultSettings(),
		updateCh:  make(chan struct{}, 1),
		releaseCh: make(chan struct{}),
	}
	eng := NewEngine(store, &fakeIdleProvider{}, &fakeLockProvider{}, nil, nil, &historyRecorderStub{})

	done := make(chan struct{})
	go func() {
		enabled := false
		if _, err := eng.UpdateSettings(settingsdomain.SettingsPatch{
			Sound: &settingsdomain.SoundSettingsPatch{Enabled: &enabled},
		}); err != nil {
			t.Errorf("UpdateSettings() err=%v", err)
		}
		close(done)
	}()

	select {
	case <-store.updateCh:
	case <-time.After(time.Second):
		t.Fatalf("expected store update to start")
	}

	readDone := make(chan struct{})
	go func() {
		_ = eng.GetRuntimeState(time.Now())
		close(readDone)
	}()

	select {
	case <-readDone:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected runtime read to proceed while settings store update is blocked")
	}

	close(store.releaseCh)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("expected UpdateSettings() to finish after store release")
	}
}

func TestEngine_NotificationConcurrencyLimitDropsExcess(t *testing.T) {
	store, err := settingsjson.OpenStore(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("OpenStore() err=%v", err)
	}
	notifier := &countingNotifierStub{
		started: make(chan struct{}, 16),
		block:   make(chan struct{}),
	}
	eng := NewEngine(store, &fakeIdleProvider{}, &fakeLockProvider{}, nil, notifier, &historyRecorderStub{})
	eng.SetReminderConfigs([]reminder.Reminder{{ID: 1, Name: "Hydrate", Enabled: true, IntervalSec: 60, BreakSec: 1, ReminderType: "notify"}})
	eng.Start(context.Background())

	for i := 0; i < notificationConcurrencyLimit+2; i++ {
		eng.notifyRemindersLocked([]int64{1}, settingsdomain.UILanguageEnUS)
	}

	timer := time.NewTimer(200 * time.Millisecond)
	defer timer.Stop()
	started := 0
loop:
	for {
		select {
		case <-notifier.started:
			started++
		case <-timer.C:
			break loop
		}
	}

	if started != notificationConcurrencyLimit {
		t.Fatalf("started=%d want=%d", started, notificationConcurrencyLimit)
	}

	close(notifier.block)
	eng.Stop()
}
