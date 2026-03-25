package engine

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"pause/internal/backend/domain/reminder"
	"pause/internal/backend/domain/settings"
	"pause/internal/backend/runtime/state"
	"pause/internal/backend/storage/settingsjson"
)

const (
	testReminderIDEye   int64 = 1
	testReminderIDStand int64 = 2
)

type fakeIdleProvider struct {
	idleSec int
}

func (f *fakeIdleProvider) CurrentIdleSeconds() int { return f.idleSec }

type fakeLockStateProvider struct {
	locked bool
}

func (f *fakeLockStateProvider) IsScreenLocked() bool { return f.locked }

type fakeHistoryRecorder struct {
	records []historyRecordCall
}

type historyRecordCall struct {
	startedAt       time.Time
	endedAt         time.Time
	source          string
	plannedBreakSec int
	actualBreakSec  int
	skipped         bool
	reminderIDs     []int64
}

func (f *fakeHistoryRecorder) RecordBreak(_ context.Context, startedAt time.Time, endedAt time.Time, source string, plannedBreakSec int, actualBreakSec int, skipped bool, reminderIDs []int64) error {
	copied := append([]int64(nil), reminderIDs...)
	f.records = append(f.records, historyRecordCall{
		startedAt:       startedAt,
		endedAt:         endedAt,
		source:          source,
		plannedBreakSec: plannedBreakSec,
		actualBreakSec:  actualBreakSec,
		skipped:         skipped,
		reminderIDs:     copied,
	})
	return nil
}

func newTestEngine(t *testing.T, idle *fakeIdleProvider) *Engine {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := settingsjson.OpenStore(path)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	engine := NewEngine(store, idle, nil, nil, nil)
	seedDefaultReminders(engine)
	return engine
}

func seedDefaultReminders(engine *Engine) {
	if engine == nil {
		return
	}
	engine.SetReminderConfigs([]reminder.Reminder{
		{ID: testReminderIDEye, Name: "Eye", Enabled: true, IntervalSec: 20 * 60, BreakSec: 20, ReminderType: "rest"},
		{ID: testReminderIDStand, Name: "Stand", Enabled: true, IntervalSec: 60 * 60, BreakSec: 5 * 60, ReminderType: "rest"},
	})
}

func reminderPatch(id int64, enabled *bool, intervalSec *int, breakSec *int) reminder.Patch {
	return reminder.Patch{
		ID:          id,
		Enabled:     enabled,
		IntervalSec: intervalSec,
		BreakSec:    breakSec,
	}
}

func setReminderPatches(t *testing.T, engine *Engine, patches ...reminder.Patch) {
	t.Helper()
	if engine == nil {
		t.Fatalf("engine is nil")
	}

	engine.mu.Lock()
	next := cloneReminderConfigs(engine.reminders)
	engine.mu.Unlock()

	for _, patch := range patches {
		id := normalizeReminderID(patch.ID)
		if id <= 0 {
			t.Fatalf("invalid reminder id: %d", patch.ID)
		}

		idx := -1
		for i := range next {
			if next[i].ID == id {
				idx = i
				break
			}
		}
		if idx < 0 {
			t.Fatalf("unknown reminder id: %d", patch.ID)
		}

		if patch.Name != nil {
			next[idx].Name = *patch.Name
		}
		if patch.Enabled != nil {
			next[idx].Enabled = *patch.Enabled
		}
		if patch.IntervalSec != nil {
			next[idx].IntervalSec = *patch.IntervalSec
		}
		if patch.BreakSec != nil {
			next[idx].BreakSec = *patch.BreakSec
		}
		if patch.ReminderType != nil {
			next[idx].ReminderType = *patch.ReminderType
		}
	}

	engine.SetReminderConfigs(next)
}

func requireReminderRuntime(t *testing.T, runtimeState state.RuntimeState, id int64) state.ReminderRuntime {
	t.Helper()
	for _, reminder := range runtimeState.Reminders {
		if reminder.ID == id {
			return reminder
		}
	}
	t.Fatalf("expected reminder runtime for id %d", id)
	return state.ReminderRuntime{}
}

func TestIdlePauseModeBoundary(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle)

	standEnabled := false
	eyeInterval := 10
	eyeBreak := 20
	setReminderPatches(t, engine,
		reminderPatch(testReminderIDStand, &standEnabled, nil, nil),
		reminderPatch(testReminderIDEye, nil, &eyeInterval, &eyeBreak),
	)

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)

	threshold := engine.GetSettings().Timer.IdlePauseThresholdSec
	idle.idleSec = threshold - 1
	for i := 1; i <= 9; i++ {
		engine.Tick(base.Add(time.Duration(i) * time.Second))
	}
	if state := engine.GetRuntimeState(base.Add(9 * time.Second)); state.CurrentSession != nil {
		t.Fatalf("did not expect session before threshold interval")
	}

	idle.idleSec = threshold
	engine.Tick(base.Add(10 * time.Second))
	if state := engine.GetRuntimeState(base.Add(10 * time.Second)); state.CurrentSession != nil {
		t.Fatalf("expected idle threshold to pause timer progression")
	}

	idle.idleSec = threshold - 1
	engine.Tick(base.Add(11 * time.Second))
	state := engine.GetRuntimeState(base.Add(11 * time.Second))
	if state.CurrentSession == nil {
		t.Fatalf("expected session after active seconds reach interval")
	}
}

func TestIdlePauseModePausesImmediatelyWhenScreenLocked(t *testing.T) {
	idle := &fakeIdleProvider{}
	lockState := &fakeLockStateProvider{}

	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := settingsjson.OpenStore(path)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	engine := NewEngine(store, idle, lockState, nil, nil)
	seedDefaultReminders(engine)

	standEnabled := false
	eyeInterval := 2
	eyeBreak := 20
	setReminderPatches(t, engine,
		reminderPatch(testReminderIDStand, &standEnabled, nil, nil),
		reminderPatch(testReminderIDEye, nil, &eyeInterval, &eyeBreak),
	)

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)

	lockState.locked = false
	engine.Tick(base.Add(1 * time.Second))
	if state := engine.GetRuntimeState(base.Add(1 * time.Second)); state.CurrentSession != nil {
		t.Fatalf("did not expect session after first active second")
	}

	lockState.locked = true
	engine.Tick(base.Add(2 * time.Second))
	if state := engine.GetRuntimeState(base.Add(2 * time.Second)); state.CurrentSession != nil {
		t.Fatalf("expected lock state to pause timer progression immediately")
	}

	lockState.locked = false
	engine.Tick(base.Add(3 * time.Second))
	state := engine.GetRuntimeState(base.Add(3 * time.Second))
	if state.CurrentSession == nil {
		t.Fatalf("expected session once unlocked active seconds resume")
	}
}

func TestRealTimeModeIgnoresIdle(t *testing.T) {
	idle := &fakeIdleProvider{idleSec: 10_000}
	engine := newTestEngine(t, idle)

	mode := settings.TimerModeRealTime
	eyeInterval := 10
	standEnabled := false
	_, err := engine.UpdateSettings(settings.SettingsPatch{
		Timer: &settings.TimerSettingsPatch{Mode: &mode},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}
	setReminderPatches(t, engine,
		reminderPatch(testReminderIDEye, nil, &eyeInterval, nil),
		reminderPatch(testReminderIDStand, &standEnabled, nil, nil),
	)

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)
	for i := 1; i <= 10; i++ {
		engine.Tick(base.Add(time.Duration(i) * time.Second))
	}

	state := engine.GetRuntimeState(base.Add(10 * time.Second))
	if state.CurrentSession == nil {
		t.Fatalf("expected session in real_time mode regardless of idle")
	}
}

func TestPauseAndResumeToggleGlobalEnabled(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle)

	eyeInterval := 5
	standEnabled := false
	setReminderPatches(t, engine,
		reminderPatch(testReminderIDEye, nil, &eyeInterval, nil),
		reminderPatch(testReminderIDStand, &standEnabled, nil, nil),
	)

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)

	state, err := engine.Pause(base)
	if err != nil {
		t.Fatalf("Pause() error = %v", err)
	}
	if state.GlobalEnabled {
		t.Fatalf("expected global reminders disabled after Pause()")
	}

	engine.Tick(base.Add(30 * time.Second))
	if state := engine.GetRuntimeState(base.Add(30 * time.Second)); state.CurrentSession != nil {
		t.Fatalf("did not expect session while global reminders disabled")
	}

	state = engine.Resume(base.Add(30 * time.Second))
	if !state.GlobalEnabled {
		t.Fatalf("expected global reminders enabled after Resume()")
	}
	for i := 31; i <= 36; i++ {
		engine.Tick(base.Add(time.Duration(i) * time.Second))
	}
	if state := engine.GetRuntimeState(base.Add(36 * time.Second)); state.CurrentSession == nil {
		t.Fatalf("expected session after resume and interval")
	}
}

func TestStandSettingChangeDoesNotResetEyeCountdown(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle)

	eyeInterval := 120
	standInterval := 3600
	setReminderPatches(t, engine,
		reminderPatch(testReminderIDEye, nil, &eyeInterval, nil),
		reminderPatch(testReminderIDStand, nil, &standInterval, nil),
	)

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)
	for i := 1; i <= 30; i++ {
		engine.Tick(base.Add(time.Duration(i) * time.Second))
	}

	before := engine.GetRuntimeState(base.Add(30 * time.Second))
	beforeEye := requireReminderRuntime(t, before, testReminderIDEye)
	if beforeEye.NextInSec <= 0 {
		t.Fatalf("expected eye countdown in progress, got %d", beforeEye.NextInSec)
	}

	standEnabled := false
	setReminderPatches(t, engine, reminderPatch(testReminderIDStand, &standEnabled, nil, nil))

	after := engine.GetRuntimeState(base.Add(30 * time.Second))
	afterEye := requireReminderRuntime(t, after, testReminderIDEye)
	if afterEye.NextInSec != beforeEye.NextInSec {
		t.Fatalf("expected eye countdown unchanged, before=%d after=%d", beforeEye.NextInSec, afterEye.NextInSec)
	}
	afterStand := requireReminderRuntime(t, after, testReminderIDStand)
	if afterStand.NextInSec != -1 {
		t.Fatalf("expected stand countdown disabled, got %d", afterStand.NextInSec)
	}
}

func TestUpdateSettingsAffectsSkipImmediately(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle)

	eyeInterval := 1
	standEnabled := false
	setReminderPatches(t, engine,
		reminderPatch(testReminderIDEye, nil, &eyeInterval, nil),
		reminderPatch(testReminderIDStand, &standEnabled, nil, nil),
	)

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)
	engine.Tick(base.Add(1 * time.Second))
	if state := engine.GetRuntimeState(base.Add(1 * time.Second)); state.CurrentSession == nil {
		t.Fatalf("expected active session")
	}

	skipAllowed := false
	_, err := engine.UpdateSettings(settings.SettingsPatch{
		Enforcement: &settings.EnforcementSettingsPatch{OverlaySkipAllowed: &skipAllowed},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}

	if _, err := engine.SkipCurrentBreak(base.Add(1*time.Second), SkipModeNormal); err == nil {
		t.Fatalf("expected skip to fail after setting changed")
	}
}

func TestSkipCurrentBreakEmergencyBypassesSkipSetting(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle)

	eyeInterval := 1
	standEnabled := false
	setReminderPatches(t, engine,
		reminderPatch(testReminderIDEye, nil, &eyeInterval, nil),
		reminderPatch(testReminderIDStand, &standEnabled, nil, nil),
	)

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)
	engine.Tick(base.Add(1 * time.Second))
	if state := engine.GetRuntimeState(base.Add(1 * time.Second)); state.CurrentSession == nil {
		t.Fatalf("expected active session")
	}

	skipAllowed := false
	_, err := engine.UpdateSettings(settings.SettingsPatch{
		Enforcement: &settings.EnforcementSettingsPatch{OverlaySkipAllowed: &skipAllowed},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}

	if _, err := engine.SkipCurrentBreak(base.Add(1*time.Second), SkipModeNormal); err == nil {
		t.Fatalf("expected normal skip to fail after setting changed")
	}

	state, err := engine.SkipCurrentBreak(base.Add(2*time.Second), SkipModeEmergency)
	if err != nil {
		t.Fatalf("SkipCurrentBreak(emergency) error = %v", err)
	}
	if state.CurrentSession != nil {
		t.Fatalf("expected session cleared after emergency skip")
	}
}

func TestSkipCurrentBreakRejectsInvalidMode(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle)

	eyeInterval := 1
	standEnabled := false
	setReminderPatches(t, engine,
		reminderPatch(testReminderIDEye, nil, &eyeInterval, nil),
		reminderPatch(testReminderIDStand, &standEnabled, nil, nil),
	)

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)
	engine.Tick(base.Add(1 * time.Second))

	if _, err := engine.SkipCurrentBreak(base.Add(2*time.Second), SkipMode("bad")); err == nil {
		t.Fatalf("expected invalid skip mode to fail")
	}
}

func TestStartBreakNow(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle)

	standEnabled := false
	eyeInterval := 1200
	eyeBreak := 25
	setReminderPatches(t, engine,
		reminderPatch(testReminderIDStand, &standEnabled, nil, nil),
		reminderPatch(testReminderIDEye, nil, &eyeInterval, &eyeBreak),
	)

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)

	state, err := engine.StartBreakNow(base.Add(1 * time.Second))
	if err != nil {
		t.Fatalf("StartBreakNow() error = %v", err)
	}
	if state.CurrentSession == nil {
		t.Fatalf("expected active session after StartBreakNow")
	}
	if state.CurrentSession.RemainingSec != eyeBreak {
		t.Fatalf("expected remaining %d, got %d", eyeBreak, state.CurrentSession.RemainingSec)
	}
	eye := requireReminderRuntime(t, state, testReminderIDEye)
	if eye.NextInSec != eyeInterval {
		t.Fatalf("expected next eye interval reset to %d, got %d", eyeInterval, eye.NextInSec)
	}
}

func TestStartBreakNowWhileGlobalDisabled(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle)

	standEnabled := false
	eyeInterval := 1200
	eyeBreak := 25
	setReminderPatches(t, engine,
		reminderPatch(testReminderIDStand, &standEnabled, nil, nil),
		reminderPatch(testReminderIDEye, nil, &eyeInterval, &eyeBreak),
	)

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)
	if _, err := engine.Pause(base); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}

	if _, err := engine.StartBreakNow(base.Add(1 * time.Second)); err == nil {
		t.Fatalf("expected StartBreakNow() to fail while global reminders disabled")
	}
}

func TestStartBreakNowForReason_ResetsOnlySelectedReminder(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle)

	eyeInterval := 1200
	eyeBreak := 20
	standInterval := 3600
	standBreak := 300
	enabled := true
	setReminderPatches(t, engine,
		reminderPatch(testReminderIDEye, &enabled, &eyeInterval, &eyeBreak),
		reminderPatch(testReminderIDStand, &enabled, &standInterval, &standBreak),
	)

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)
	engine.Tick(base.Add(600 * time.Second))

	state, err := engine.StartBreakNowForReason(testReminderIDEye, base.Add(601*time.Second))
	if err != nil {
		t.Fatalf("StartBreakNowForReason() error = %v", err)
	}
	if state.CurrentSession == nil {
		t.Fatalf("expected active session after StartBreakNowForReason")
	}
	eye := requireReminderRuntime(t, state, testReminderIDEye)
	if eye.NextInSec != eyeInterval {
		t.Fatalf("expected eye countdown reset to %d, got %d", eyeInterval, eye.NextInSec)
	}
	stand := requireReminderRuntime(t, state, testReminderIDStand)
	if stand.NextInSec != 3000 {
		t.Fatalf("expected stand countdown unchanged at 3000, got %d", stand.NextInSec)
	}
}

func TestPauseReminder_AffectsOnlySelectedReminder(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle)
	setReminderPatches(t, engine,
		reminderPatch(testReminderIDEye, nil, nil, nil),
		reminderPatch(testReminderIDStand, nil, nil, nil),
	)

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)
	engine.Tick(base.Add(120 * time.Second))

	state, err := engine.PauseReminder(testReminderIDEye, base.Add(121*time.Second))
	if err != nil {
		t.Fatalf("PauseReminder() error = %v", err)
	}
	eye := requireReminderRuntime(t, state, testReminderIDEye)
	stand := requireReminderRuntime(t, state, testReminderIDStand)
	if !eye.Paused || stand.Paused {
		t.Fatalf("expected only eye reminder paused, got eye=%t stand=%t", eye.Paused, stand.Paused)
	}
	if eye.NextInSec != -1 {
		t.Fatalf("expected eye countdown hidden while paused, got %d", eye.NextInSec)
	}
	if stand.NextInSec < 0 {
		t.Fatalf("expected stand countdown still active, got %d", stand.NextInSec)
	}

	state, err = engine.ResumeReminder(testReminderIDEye, base.Add(122*time.Second))
	if err != nil {
		t.Fatalf("ResumeReminder() error = %v", err)
	}
	eye = requireReminderRuntime(t, state, testReminderIDEye)
	if eye.Paused {
		t.Fatalf("expected eye reminder resumed")
	}
	if eye.NextInSec < 0 {
		t.Fatalf("expected eye countdown active after resume, got %d", eye.NextInSec)
	}
}

func TestHistoryRecorder_ManualBreakLifecycle(t *testing.T) {
	idle := &fakeIdleProvider{}
	history := &fakeHistoryRecorder{}
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := settingsjson.OpenStore(path)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	engine := NewEngine(store, idle, nil, nil, history)
	seedDefaultReminders(engine)

	standEnabled := false
	eyeInterval := 1200
	eyeBreak := 5
	setReminderPatches(t, engine,
		reminderPatch(testReminderIDStand, &standEnabled, nil, nil),
		reminderPatch(testReminderIDEye, nil, &eyeInterval, &eyeBreak),
	)

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)
	state, err := engine.StartBreakNow(base.Add(1 * time.Second))
	if err != nil {
		t.Fatalf("StartBreakNow() error = %v", err)
	}
	if state.CurrentSession == nil {
		t.Fatalf("expected active session")
	}
	if len(history.records) != 0 {
		t.Fatalf("expected no history records before break ends, got %d", len(history.records))
	}

	engine.Tick(base.Add(6 * time.Second))
	if len(history.records) != 1 {
		t.Fatalf("expected 1 history record call, got %d", len(history.records))
	}
	if history.records[0].source != "manual" {
		t.Fatalf("expected manual source, got %q", history.records[0].source)
	}
	if history.records[0].plannedBreakSec != eyeBreak {
		t.Fatalf("expected planned break sec %d, got %d", eyeBreak, history.records[0].plannedBreakSec)
	}
	if history.records[0].actualBreakSec != eyeBreak {
		t.Fatalf("expected actual break sec %d, got %d", eyeBreak, history.records[0].actualBreakSec)
	}
	if history.records[0].skipped {
		t.Fatalf("expected completed record, got skipped=true")
	}
}

func TestHistoryRecorder_SkipBreak(t *testing.T) {
	idle := &fakeIdleProvider{}
	history := &fakeHistoryRecorder{}
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := settingsjson.OpenStore(path)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	engine := NewEngine(store, idle, nil, nil, history)
	seedDefaultReminders(engine)

	standEnabled := false
	eyeInterval := 1200
	eyeBreak := 60
	setReminderPatches(t, engine,
		reminderPatch(testReminderIDStand, &standEnabled, nil, nil),
		reminderPatch(testReminderIDEye, nil, &eyeInterval, &eyeBreak),
	)

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)
	if _, err := engine.StartBreakNow(base.Add(1 * time.Second)); err != nil {
		t.Fatalf("StartBreakNow() error = %v", err)
	}
	if _, err := engine.SkipCurrentBreak(base.Add(11*time.Second), SkipModeNormal); err != nil {
		t.Fatalf("SkipCurrentBreak() error = %v", err)
	}

	if len(history.records) != 1 {
		t.Fatalf("expected 1 history record call, got %d", len(history.records))
	}
	if history.records[0].actualBreakSec != 10 {
		t.Fatalf("expected skip actual sec 10, got %d", history.records[0].actualBreakSec)
	}
	if !history.records[0].skipped {
		t.Fatalf("expected skipped record, got skipped=false")
	}
}

func TestSetReminderConfigsAllowsEmptySnapshot(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle)

	base := time.Unix(1_700_000_000, 0)
	engine.SetReminderConfigs(nil)

	state := engine.GetRuntimeState(base)
	if len(state.Reminders) != 0 {
		t.Fatalf("expected empty reminders snapshot, got %d", len(state.Reminders))
	}
	if len(state.NextBreakReason) != 0 {
		t.Fatalf("expected no next break reasons, got %d", len(state.NextBreakReason))
	}
	if _, err := engine.StartBreakNow(base.Add(time.Second)); err == nil {
		t.Fatalf("expected StartBreakNow to fail when reminders snapshot is empty")
	}
}
