package service

import (
	"path/filepath"
	"testing"
	"time"

	"pause/internal/config"
)

type fakeIdleProvider struct {
	idleSec int
}

func (f *fakeIdleProvider) CurrentIdleSeconds() int { return f.idleSec }

type fakeStartupManager struct {
	lastValue bool
	calls     int
	current   bool
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

func newTestEngine(t *testing.T, idle *fakeIdleProvider, startup *fakeStartupManager) *Engine {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := config.NewStore(path)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	return NewEngine(store, idle, nil, startup)
}

func TestIdlePauseModeBoundary(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle, &fakeStartupManager{})

	standEnabled := false
	eyeInterval := 10
	eyeBreak := 20
	_, err := engine.UpdateSettings(config.SettingsPatch{
		Stand: &config.ReminderRulePatch{Enabled: &standEnabled},
		Eye:   &config.ReminderRulePatch{IntervalSec: &eyeInterval, BreakSec: &eyeBreak},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)

	idle.idleSec = 299
	for i := 1; i <= 9; i++ {
		engine.Tick(base.Add(time.Duration(i) * time.Second))
	}
	if state := engine.GetRuntimeState(base.Add(9 * time.Second)); state.CurrentSession != nil {
		t.Fatalf("did not expect session before threshold interval")
	}

	idle.idleSec = 300
	engine.Tick(base.Add(10 * time.Second))
	if state := engine.GetRuntimeState(base.Add(10 * time.Second)); state.CurrentSession != nil {
		t.Fatalf("expected idle threshold to pause timer progression")
	}

	idle.idleSec = 299
	engine.Tick(base.Add(11 * time.Second))
	state := engine.GetRuntimeState(base.Add(11 * time.Second))
	if state.CurrentSession == nil {
		t.Fatalf("expected session after active seconds reach interval")
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
		Eye:   &config.ReminderRulePatch{IntervalSec: &eyeInterval},
		Stand: &config.ReminderRulePatch{Enabled: &standEnabled},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}

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

func TestPauseAndResume(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle, &fakeStartupManager{})

	eyeInterval := 5
	standEnabled := false
	_, err := engine.UpdateSettings(config.SettingsPatch{
		Eye:   &config.ReminderRulePatch{IntervalSec: &eyeInterval},
		Stand: &config.ReminderRulePatch{Enabled: &standEnabled},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)

	if _, err := engine.Pause(PauseModeTemporary, 60, base); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}

	engine.Tick(base.Add(30 * time.Second))
	if state := engine.GetRuntimeState(base.Add(30 * time.Second)); state.CurrentSession != nil {
		t.Fatalf("did not expect session while paused")
	}

	engine.Resume(base.Add(30 * time.Second))
	for i := 31; i <= 35; i++ {
		engine.Tick(base.Add(time.Duration(i) * time.Second))
	}
	if state := engine.GetRuntimeState(base.Add(35 * time.Second)); state.CurrentSession == nil {
		t.Fatalf("expected session after resume and interval")
	}
}

func TestStandSettingChangeDoesNotResetEyeCountdown(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle, &fakeStartupManager{})

	eyeInterval := 120
	standInterval := 3600
	_, err := engine.UpdateSettings(config.SettingsPatch{
		Eye:   &config.ReminderRulePatch{IntervalSec: &eyeInterval},
		Stand: &config.ReminderRulePatch{IntervalSec: &standInterval},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)
	for i := 1; i <= 30; i++ {
		engine.Tick(base.Add(time.Duration(i) * time.Second))
	}

	before := engine.GetRuntimeState(base.Add(30 * time.Second))
	if before.NextEyeInSec <= 0 {
		t.Fatalf("expected eye countdown in progress, got %d", before.NextEyeInSec)
	}

	standEnabled := false
	_, err = engine.UpdateSettings(config.SettingsPatch{
		Stand: &config.ReminderRulePatch{Enabled: &standEnabled},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}

	after := engine.GetRuntimeState(base.Add(30 * time.Second))
	if after.NextEyeInSec != before.NextEyeInSec {
		t.Fatalf("expected eye countdown unchanged, before=%d after=%d", before.NextEyeInSec, after.NextEyeInSec)
	}
	if after.NextStandInSec != -1 {
		t.Fatalf("expected stand countdown disabled, got %d", after.NextStandInSec)
	}
}

func TestUpdateSettingsAffectsSkipImmediately(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle, &fakeStartupManager{})

	eyeInterval := 1
	standEnabled := false
	_, err := engine.UpdateSettings(config.SettingsPatch{
		Eye:   &config.ReminderRulePatch{IntervalSec: &eyeInterval},
		Stand: &config.ReminderRulePatch{Enabled: &standEnabled},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)
	engine.Tick(base.Add(1 * time.Second))
	if state := engine.GetRuntimeState(base.Add(1 * time.Second)); state.CurrentSession == nil {
		t.Fatalf("expected active session")
	}

	skipAllowed := false
	_, err = engine.UpdateSettings(config.SettingsPatch{
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
	_, err := engine.UpdateSettings(config.SettingsPatch{
		Eye:   &config.ReminderRulePatch{IntervalSec: &eyeInterval},
		Stand: &config.ReminderRulePatch{Enabled: &standEnabled},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)
	engine.Tick(base.Add(1 * time.Second))
	if state := engine.GetRuntimeState(base.Add(1 * time.Second)); state.CurrentSession == nil {
		t.Fatalf("expected active session")
	}

	skipAllowed := false
	_, err = engine.UpdateSettings(config.SettingsPatch{
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
	_, err := engine.UpdateSettings(config.SettingsPatch{
		Eye:   &config.ReminderRulePatch{IntervalSec: &eyeInterval},
		Stand: &config.ReminderRulePatch{Enabled: &standEnabled},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}

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
	_, err := engine.UpdateSettings(config.SettingsPatch{
		Stand: &config.ReminderRulePatch{Enabled: &standEnabled},
		Eye:   &config.ReminderRulePatch{IntervalSec: &eyeInterval, BreakSec: &eyeBreak},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}

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
	if state.NextEyeInSec != eyeInterval {
		t.Fatalf("expected next eye interval reset to %d, got %d", eyeInterval, state.NextEyeInSec)
	}
}

func TestStartBreakNowWhilePaused(t *testing.T) {
	idle := &fakeIdleProvider{}
	engine := newTestEngine(t, idle, &fakeStartupManager{})

	standEnabled := false
	eyeInterval := 1200
	eyeBreak := 25
	_, err := engine.UpdateSettings(config.SettingsPatch{
		Stand: &config.ReminderRulePatch{Enabled: &standEnabled},
		Eye:   &config.ReminderRulePatch{IntervalSec: &eyeInterval, BreakSec: &eyeBreak},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}

	base := time.Unix(1_700_000_000, 0)
	engine.Tick(base)
	if _, err := engine.Pause(PauseModeIndefinite, 0, base); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}

	state, err := engine.StartBreakNow(base.Add(1 * time.Second))
	if err != nil {
		t.Fatalf("StartBreakNow() error = %v", err)
	}
	if state.Paused {
		t.Fatalf("expected paused to be false after StartBreakNow")
	}
	if state.CurrentSession == nil {
		t.Fatalf("expected active session after StartBreakNow while paused")
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
	engine1 := NewEngine(store1, idle, nil, startup)
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
	engine2 := NewEngine(store2, idle, nil, startup)
	if err := engine2.SyncPlatformSettings(); err != nil {
		t.Fatalf("second SyncPlatformSettings() error = %v", err)
	}
	if startup.calls != 0 {
		t.Fatalf("expected existing config startup to read system state without re-applying")
	}
}
