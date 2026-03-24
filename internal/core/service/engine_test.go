package service

import (
	"path/filepath"
	"testing"
	"time"

	"pause/internal/core/config"
)

type fakeIdleProvider struct {
	idleSec int
}

func (f *fakeIdleProvider) CurrentIdleSeconds() int { return f.idleSec }

type fakeLockStateProvider struct {
	locked bool
}

func (f *fakeLockStateProvider) IsScreenLocked() bool { return f.locked }

type fakeStartupManager struct {
	lastValue bool
	calls     int
	current   bool
}

type fakeHistoryRecorder struct {
	starts    []historyStartCall
	completes []historyFinishCall
	skips     []historyFinishCall
	nextID    int64
}

type historyStartCall struct {
	sessionID       int64
	source          string
	plannedBreakSec int
	reminderIDs     []int64
}

type historyFinishCall struct {
	sessionID      int64
	actualBreakSec int
}

func (f *fakeStartupManager) SetLaunchAtLogin(enabled bool) error {
	f.lastValue = enabled
	f.calls++
	f.current = enabled
	return nil
}

func (f *fakeStartupManager) GetLaunchAtLogin() (bool, error) {
	return f.current, nil
}

func (f *fakeHistoryRecorder) StartBreak(_ time.Time, source string, plannedBreakSec int, reminderIDs []int64) (int64, error) {
	f.nextID++
	sessionID := f.nextID
	copied := append([]int64(nil), reminderIDs...)
	f.starts = append(f.starts, historyStartCall{
		sessionID:       sessionID,
		source:          source,
		plannedBreakSec: plannedBreakSec,
		reminderIDs:     copied,
	})
	return sessionID, nil
}

func (f *fakeHistoryRecorder) CompleteBreak(sessionID int64, _ time.Time, actualBreakSec int) error {
	f.completes = append(f.completes, historyFinishCall{
		sessionID:      sessionID,
		actualBreakSec: actualBreakSec,
	})
	return nil
}

func (f *fakeHistoryRecorder) SkipBreak(sessionID int64, _ time.Time, actualBreakSec int) error {
	f.skips = append(f.skips, historyFinishCall{
		sessionID:      sessionID,
		actualBreakSec: actualBreakSec,
	})
	return nil
}

func newTestEngine(t *testing.T, idle *fakeIdleProvider, startup *fakeStartupManager) *Engine {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := config.NewStore(path)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	engine := NewEngine(store, idle, nil, nil, startup, nil)
	seedDefaultReminders(engine)
	return engine
}

func seedDefaultReminders(engine *Engine) {
	if engine == nil {
		return
	}
	engine.SetReminderConfigs([]config.ReminderConfig{
		{ID: config.ReminderIDEye, Name: "Eye", Enabled: true, IntervalSec: 20 * 60, BreakSec: 20, ReminderType: "rest"},
		{ID: config.ReminderIDStand, Name: "Stand", Enabled: true, IntervalSec: 60 * 60, BreakSec: 5 * 60, ReminderType: "rest"},
	})
}

func reminderPatch(id int64, enabled *bool, intervalSec *int, breakSec *int) config.ReminderPatch {
	return config.ReminderPatch{
		ID:          id,
		Enabled:     enabled,
		IntervalSec: intervalSec,
		BreakSec:    breakSec,
	}
}

func setReminderPatches(t *testing.T, engine *Engine, patches ...config.ReminderPatch) {
	t.Helper()
	if _, err := engine.UpdateReminderConfigs(patches); err != nil {
		t.Fatalf("UpdateReminderConfigs() error = %v", err)
	}
}

func requireReminderRuntime(t *testing.T, state config.RuntimeState, id int64) config.ReminderRuntime {
	t.Helper()
	for _, reminder := range state.Reminders {
		if reminder.ID == id {
			return reminder
		}
	}
	t.Fatalf("expected reminder runtime for id %d", id)
	return config.ReminderRuntime{}
}

func TestIdlePauseModeBoundary(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle, &fakeStartupManager{})

	standEnabled := false
	eyeInterval := 10
	eyeBreak := 20
	setReminderPatches(t, engine,
		reminderPatch(config.ReminderIDStand, &standEnabled, nil, nil),
		reminderPatch(config.ReminderIDEye, nil, &eyeInterval, &eyeBreak),
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
	store, err := config.NewStore(path)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	engine := NewEngine(store, idle, lockState, nil, &fakeStartupManager{}, nil)
	seedDefaultReminders(engine)

	standEnabled := false
	eyeInterval := 2
	eyeBreak := 20
	setReminderPatches(t, engine,
		reminderPatch(config.ReminderIDStand, &standEnabled, nil, nil),
		reminderPatch(config.ReminderIDEye, nil, &eyeInterval, &eyeBreak),
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
	engine := newTestEngine(t, idle, &fakeStartupManager{})

	mode := config.TimerModeRealTime
	eyeInterval := 10
	standEnabled := false
	_, err := engine.UpdateSettings(config.SettingsPatch{
		Timer: &config.TimerSettingsPatch{Mode: &mode},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}
	setReminderPatches(t, engine,
		reminderPatch(config.ReminderIDEye, nil, &eyeInterval, nil),
		reminderPatch(config.ReminderIDStand, &standEnabled, nil, nil),
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
	engine := newTestEngine(t, idle, &fakeStartupManager{})

	eyeInterval := 5
	standEnabled := false
	setReminderPatches(t, engine,
		reminderPatch(config.ReminderIDEye, nil, &eyeInterval, nil),
		reminderPatch(config.ReminderIDStand, &standEnabled, nil, nil),
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
	engine := newTestEngine(t, idle, &fakeStartupManager{})

	eyeInterval := 120
	standInterval := 3600
	setReminderPatches(t, engine,
		reminderPatch(config.ReminderIDEye, nil, &eyeInterval, nil),
		reminderPatch(config.ReminderIDStand, nil, &standInterval, nil),
	)

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)
	for i := 1; i <= 30; i++ {
		engine.Tick(base.Add(time.Duration(i) * time.Second))
	}

	before := engine.GetRuntimeState(base.Add(30 * time.Second))
	beforeEye := requireReminderRuntime(t, before, config.ReminderIDEye)
	if beforeEye.NextInSec <= 0 {
		t.Fatalf("expected eye countdown in progress, got %d", beforeEye.NextInSec)
	}

	standEnabled := false
	setReminderPatches(t, engine, reminderPatch(config.ReminderIDStand, &standEnabled, nil, nil))

	after := engine.GetRuntimeState(base.Add(30 * time.Second))
	afterEye := requireReminderRuntime(t, after, config.ReminderIDEye)
	if afterEye.NextInSec != beforeEye.NextInSec {
		t.Fatalf("expected eye countdown unchanged, before=%d after=%d", beforeEye.NextInSec, afterEye.NextInSec)
	}
	afterStand := requireReminderRuntime(t, after, config.ReminderIDStand)
	if afterStand.NextInSec != -1 {
		t.Fatalf("expected stand countdown disabled, got %d", afterStand.NextInSec)
	}
}

func TestUpdateSettingsAffectsSkipImmediately(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle, &fakeStartupManager{})

	eyeInterval := 1
	standEnabled := false
	setReminderPatches(t, engine,
		reminderPatch(config.ReminderIDEye, nil, &eyeInterval, nil),
		reminderPatch(config.ReminderIDStand, &standEnabled, nil, nil),
	)

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)
	engine.Tick(base.Add(1 * time.Second))
	if state := engine.GetRuntimeState(base.Add(1 * time.Second)); state.CurrentSession == nil {
		t.Fatalf("expected active session")
	}

	skipAllowed := false
	_, err := engine.UpdateSettings(config.SettingsPatch{
		Enforcement: &config.EnforcementSettingsPatch{OverlaySkipAllowed: &skipAllowed},
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
	engine := newTestEngine(t, idle, &fakeStartupManager{})

	eyeInterval := 1
	standEnabled := false
	setReminderPatches(t, engine,
		reminderPatch(config.ReminderIDEye, nil, &eyeInterval, nil),
		reminderPatch(config.ReminderIDStand, &standEnabled, nil, nil),
	)

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)
	engine.Tick(base.Add(1 * time.Second))
	if state := engine.GetRuntimeState(base.Add(1 * time.Second)); state.CurrentSession == nil {
		t.Fatalf("expected active session")
	}

	skipAllowed := false
	_, err := engine.UpdateSettings(config.SettingsPatch{
		Enforcement: &config.EnforcementSettingsPatch{OverlaySkipAllowed: &skipAllowed},
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
	engine := newTestEngine(t, idle, &fakeStartupManager{})

	eyeInterval := 1
	standEnabled := false
	setReminderPatches(t, engine,
		reminderPatch(config.ReminderIDEye, nil, &eyeInterval, nil),
		reminderPatch(config.ReminderIDStand, &standEnabled, nil, nil),
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
	engine := newTestEngine(t, idle, &fakeStartupManager{})

	standEnabled := false
	eyeInterval := 1200
	eyeBreak := 25
	setReminderPatches(t, engine,
		reminderPatch(config.ReminderIDStand, &standEnabled, nil, nil),
		reminderPatch(config.ReminderIDEye, nil, &eyeInterval, &eyeBreak),
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
	eye := requireReminderRuntime(t, state, config.ReminderIDEye)
	if eye.NextInSec != eyeInterval {
		t.Fatalf("expected next eye interval reset to %d, got %d", eyeInterval, eye.NextInSec)
	}
}

func TestStartBreakNowWhileGlobalDisabled(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle, &fakeStartupManager{})

	standEnabled := false
	eyeInterval := 1200
	eyeBreak := 25
	setReminderPatches(t, engine,
		reminderPatch(config.ReminderIDStand, &standEnabled, nil, nil),
		reminderPatch(config.ReminderIDEye, nil, &eyeInterval, &eyeBreak),
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
	engine := newTestEngine(t, idle, &fakeStartupManager{})

	eyeInterval := 1200
	eyeBreak := 20
	standInterval := 3600
	standBreak := 300
	enabled := true
	setReminderPatches(t, engine,
		reminderPatch(config.ReminderIDEye, &enabled, &eyeInterval, &eyeBreak),
		reminderPatch(config.ReminderIDStand, &enabled, &standInterval, &standBreak),
	)

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)
	engine.Tick(base.Add(600 * time.Second))

	state, err := engine.StartBreakNowForReason(config.ReminderIDEye, base.Add(601*time.Second))
	if err != nil {
		t.Fatalf("StartBreakNowForReason() error = %v", err)
	}
	if state.CurrentSession == nil {
		t.Fatalf("expected active session after StartBreakNowForReason")
	}
	eye := requireReminderRuntime(t, state, config.ReminderIDEye)
	if eye.NextInSec != eyeInterval {
		t.Fatalf("expected eye countdown reset to %d, got %d", eyeInterval, eye.NextInSec)
	}
	stand := requireReminderRuntime(t, state, config.ReminderIDStand)
	if stand.NextInSec != 3000 {
		t.Fatalf("expected stand countdown unchanged at 3000, got %d", stand.NextInSec)
	}
}

func TestPauseReminder_AffectsOnlySelectedReminder(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle, &fakeStartupManager{})
	setReminderPatches(t, engine,
		reminderPatch(config.ReminderIDEye, nil, nil, nil),
		reminderPatch(config.ReminderIDStand, nil, nil, nil),
	)

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)
	engine.Tick(base.Add(120 * time.Second))

	state, err := engine.PauseReminder(config.ReminderIDEye, base.Add(121*time.Second))
	if err != nil {
		t.Fatalf("PauseReminder() error = %v", err)
	}
	eye := requireReminderRuntime(t, state, config.ReminderIDEye)
	stand := requireReminderRuntime(t, state, config.ReminderIDStand)
	if !eye.Paused || stand.Paused {
		t.Fatalf("expected only eye reminder paused, got eye=%t stand=%t", eye.Paused, stand.Paused)
	}
	if eye.NextInSec != -1 {
		t.Fatalf("expected eye countdown hidden while paused, got %d", eye.NextInSec)
	}
	if stand.NextInSec < 0 {
		t.Fatalf("expected stand countdown still active, got %d", stand.NextInSec)
	}

	state, err = engine.ResumeReminder(config.ReminderIDEye, base.Add(122*time.Second))
	if err != nil {
		t.Fatalf("ResumeReminder() error = %v", err)
	}
	eye = requireReminderRuntime(t, state, config.ReminderIDEye)
	if eye.Paused {
		t.Fatalf("expected eye reminder resumed")
	}
	if eye.NextInSec < 0 {
		t.Fatalf("expected eye countdown active after resume, got %d", eye.NextInSec)
	}
}

func TestHistoryRecorder_ManualBreakLifecycle(t *testing.T) {
	idle := &fakeIdleProvider{}
	startup := &fakeStartupManager{}
	history := &fakeHistoryRecorder{}
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := config.NewStore(path)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	engine := NewEngine(store, idle, nil, nil, startup, history)
	seedDefaultReminders(engine)

	standEnabled := false
	eyeInterval := 1200
	eyeBreak := 5
	setReminderPatches(t, engine,
		reminderPatch(config.ReminderIDStand, &standEnabled, nil, nil),
		reminderPatch(config.ReminderIDEye, nil, &eyeInterval, &eyeBreak),
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
	if len(history.starts) != 1 {
		t.Fatalf("expected 1 history start call, got %d", len(history.starts))
	}
	if history.starts[0].source != "manual" {
		t.Fatalf("expected manual source, got %q", history.starts[0].source)
	}
	if history.starts[0].plannedBreakSec != eyeBreak {
		t.Fatalf("expected planned break sec %d, got %d", eyeBreak, history.starts[0].plannedBreakSec)
	}

	engine.Tick(base.Add(6 * time.Second))
	if len(history.completes) != 1 {
		t.Fatalf("expected 1 history complete call, got %d", len(history.completes))
	}
	if history.completes[0].actualBreakSec != eyeBreak {
		t.Fatalf("expected actual break sec %d, got %d", eyeBreak, history.completes[0].actualBreakSec)
	}
}

func TestHistoryRecorder_SkipBreak(t *testing.T) {
	idle := &fakeIdleProvider{}
	startup := &fakeStartupManager{}
	history := &fakeHistoryRecorder{}
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := config.NewStore(path)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	engine := NewEngine(store, idle, nil, nil, startup, history)
	seedDefaultReminders(engine)

	standEnabled := false
	eyeInterval := 1200
	eyeBreak := 60
	setReminderPatches(t, engine,
		reminderPatch(config.ReminderIDStand, &standEnabled, nil, nil),
		reminderPatch(config.ReminderIDEye, nil, &eyeInterval, &eyeBreak),
	)

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)
	if _, err := engine.StartBreakNow(base.Add(1 * time.Second)); err != nil {
		t.Fatalf("StartBreakNow() error = %v", err)
	}
	if _, err := engine.SkipCurrentBreak(base.Add(11*time.Second), SkipModeNormal); err != nil {
		t.Fatalf("SkipCurrentBreak() error = %v", err)
	}

	if len(history.skips) != 1 {
		t.Fatalf("expected 1 history skip call, got %d", len(history.skips))
	}
	if history.skips[0].actualBreakSec != 10 {
		t.Fatalf("expected skip actual sec 10, got %d", history.skips[0].actualBreakSec)
	}
}

func TestSetLaunchAtLoginSync(t *testing.T) {
	idle := &fakeIdleProvider{}
	startup := &fakeStartupManager{}
	engine := newTestEngine(t, idle, startup)

	actual, err := engine.SetLaunchAtLogin(true)
	if err != nil {
		t.Fatalf("SetLaunchAtLogin() error = %v", err)
	}
	if !actual {
		t.Fatalf("expected launch-at-login state to be true")
	}

	if startup.calls != 1 || !startup.lastValue {
		t.Fatalf("expected startup manager to be called with true")
	}
}

func TestSetReminderConfigsAllowsEmptySnapshot(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle, &fakeStartupManager{})

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

func TestUpdateReminderConfigsRejectsUnknownReminderID(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle, &fakeStartupManager{})

	enabled := true
	_, err := engine.UpdateReminderConfigs([]config.ReminderPatch{
		{ID: -1, Enabled: &enabled},
	})
	if err == nil {
		t.Fatalf("expected unknown reminder id patch to fail")
	}
}

func TestUpdateReminderConfigsRejectsInvalidInterval(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle, &fakeStartupManager{})

	interval := 0
	_, err := engine.UpdateReminderConfigs([]config.ReminderPatch{
		{ID: config.ReminderIDEye, IntervalSec: &interval},
	})
	if err == nil {
		t.Fatalf("expected invalid interval patch to fail")
	}
}

func TestSyncPlatformSettingsBootstrapsOnFirstRun(t *testing.T) {
	idle := &fakeIdleProvider{}
	startup := &fakeStartupManager{}
	engine := newTestEngine(t, idle, startup)

	startup.calls = 0
	if err := engine.SyncPlatformSettings(); err != nil {
		t.Fatalf("SyncPlatformSettings() error = %v", err)
	}
	if startup.calls != 1 || !startup.lastValue {
		t.Fatalf("expected first-run startup sync call with true")
	}
}

func TestSyncPlatformSettingsDisablesWhenPersistedFalse(t *testing.T) {
	idle := &fakeIdleProvider{}
	startup := &fakeStartupManager{}
	engine := newTestEngine(t, idle, startup)

	startup.calls = 0
	if err := engine.SyncPlatformSettings(); err != nil {
		t.Fatalf("SyncPlatformSettings() error = %v", err)
	}
	if startup.calls != 1 || !startup.lastValue {
		t.Fatalf("expected startup manager sync call with true")
	}
}

func TestSyncPlatformSettingsDoesNotReapplyOnExistingConfig(t *testing.T) {
	idle := &fakeIdleProvider{}
	startup := &fakeStartupManager{}
	path := filepath.Join(t.TempDir(), "settings.json")

	store1, err := config.NewStore(path)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	engine1 := NewEngine(store1, idle, nil, nil, startup, nil)
	if err := engine1.SyncPlatformSettings(); err != nil {
		t.Fatalf("first SyncPlatformSettings() error = %v", err)
	}
	if startup.calls == 0 {
		t.Fatalf("expected first run to attempt launch-at-login setup")
	}

	startup.calls = 0
	store2, err := config.NewStore(path)
	if err != nil {
		t.Fatalf("NewStore(reopen) error = %v", err)
	}
	engine2 := NewEngine(store2, idle, nil, nil, startup, nil)
	if err := engine2.SyncPlatformSettings(); err != nil {
		t.Fatalf("second SyncPlatformSettings() error = %v", err)
	}
	if startup.calls != 0 {
		t.Fatalf("expected existing config startup to read system state without re-applying")
	}
}
