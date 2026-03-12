package service

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"time"

	"pause/internal/config"
	"pause/internal/diag"
	"pause/internal/platform"
	"pause/internal/scheduler"
	"pause/internal/session"
)

const (
	PauseModeTemporary  = "temporary"
	PauseModeIndefinite = "indefinite"
)

type Engine struct {
	mu        sync.Mutex
	startOnce sync.Once

	store     *config.Store
	scheduler *scheduler.Scheduler
	session   *session.Manager

	idleProvider   platform.IdleProvider
	notifier       platform.Notifier
	soundPlayer    platform.SoundPlayer
	startupManager platform.StartupManager

	lastTick      time.Time
	tickRemainder time.Duration

	paused      bool
	pauseMode   string
	pausedUntil *time.Time

	lastTickActive bool
	currentIdleSec int
}

func NewEngine(
	store *config.Store,
	idleProvider platform.IdleProvider,
	notifier platform.Notifier,
	soundPlayer platform.SoundPlayer,
	startupManager platform.StartupManager,
) *Engine {
	if idleProvider == nil {
		idleProvider = platform.NoopIdleProvider{}
	}
	if notifier == nil {
		notifier = platform.NoopNotifier{}
	}
	if soundPlayer == nil {
		soundPlayer = platform.NoopSoundPlayer{}
	}
	if startupManager == nil {
		startupManager = platform.NoopStartupManager{}
	}

	return &Engine{
		store:          store,
		scheduler:      scheduler.New(),
		session:        session.NewManager(),
		idleProvider:   idleProvider,
		notifier:       notifier,
		soundPlayer:    soundPlayer,
		startupManager: startupManager,
	}
}

func (e *Engine) SyncPlatformSettings() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	settings := e.store.Get()
	// Do not re-apply launch-at-login "enabled" on every app startup.
	// On macOS this can trigger launchd reload while the app is booting,
	// causing slow startup and potential self-termination.
	if settings.Startup.LaunchAtLogin {
		return nil
	}
	return e.startupManager.SetLaunchAtLogin(false)
}

func (e *Engine) Start(ctx context.Context) {
	e.startOnce.Do(func() {
		diag.Logf("engine.start")
		ticker := time.NewTicker(time.Second)
		go func() {
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case now := <-ticker.C:
					e.Tick(now)
				}
			}
		}()
	})
}

func (e *Engine) Tick(now time.Time) {
	idleSec := e.idleProvider.CurrentIdleSeconds()

	e.mu.Lock()
	defer e.mu.Unlock()

	firstTick := e.lastTick.IsZero()
	if firstTick {
		e.lastTick = now
	}

	settings := e.store.Get()
	e.currentIdleSec = idleSec
	e.lastTickActive = e.isTickActive(settings)
	rawDeltaSec := int(now.Sub(e.lastTick).Seconds())
	appliedDeltaSec := rawDeltaSec
	if firstTick {
		e.logTickLocked(now, settings, "bootstrap", 0, 0, nil)
		return
	}

	e.session.Tick(now)
	if view := e.session.CurrentView(now); view != nil && view.Status == string(session.StatusCompleted) {
		_ = e.soundPlayer.PlayBreakEnd(settings.Sound)
	}
	e.session.ClearIfDone()

	if !settings.GlobalEnabled {
		e.lastTick = now
		e.tickRemainder = 0
		e.logTickLocked(now, settings, "global_disabled", rawDeltaSec, appliedDeltaSec, nil)
		return
	}

	if e.syncPause(now) {
		e.lastTick = now
		e.tickRemainder = 0
		e.logTickLocked(now, settings, "paused", rawDeltaSec, appliedDeltaSec, nil)
		return
	}

	if e.session.IsActive() {
		e.lastTick = now
		e.tickRemainder = 0
		e.logTickLocked(now, settings, "session_active", rawDeltaSec, appliedDeltaSec, nil)
		return
	}

	if !e.lastTickActive {
		e.lastTick = now
		e.tickRemainder = 0
		e.logTickLocked(now, settings, "idle_paused", rawDeltaSec, appliedDeltaSec, nil)
		return
	}

	elapsed := now.Sub(e.lastTick) + e.tickRemainder
	if elapsed < 0 {
		e.lastTick = now
		e.tickRemainder = 0
		e.logTickLocked(now, settings, "negative_elapsed", rawDeltaSec, 0, nil)
		return
	}

	rawDeltaSec = int(elapsed / time.Second)
	if rawDeltaSec <= 0 {
		e.lastTick = now
		e.tickRemainder = elapsed
		e.logTickLocked(now, settings, "sub_second_elapsed", rawDeltaSec, 0, nil)
		return
	}

	appliedDeltaSec = rawDeltaSec
	e.tickRemainder = elapsed - (time.Duration(appliedDeltaSec) * time.Second)

	evt := e.scheduler.OnActiveSeconds(appliedDeltaSec, settings)
	e.lastTick = now
	if evt == nil {
		e.logTickLocked(now, settings, "no_event", rawDeltaSec, appliedDeltaSec, nil)
		return
	}

	e.session.StartBreak(now, evt, settings.Enforcement.OverlaySkipAllowed)
	_ = e.notifier.ShowReminder("Time to rest", buildReasonText(evt.Reasons, evt.BreakSec))
	e.logTickLocked(now, settings, "event", rawDeltaSec, appliedDeltaSec, evt)
}

func (e *Engine) GetSettings() config.Settings {
	return e.store.Get()
}

func (e *Engine) UpdateSettings(patch config.SettingsPatch) (config.Settings, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	prev := e.store.Get()
	next, err := e.store.Update(patch)
	if err != nil {
		return config.Settings{}, err
	}

	if patch.Startup != nil && patch.Startup.LaunchAtLogin != nil {
		if err := e.startupManager.SetLaunchAtLogin(next.Startup.LaunchAtLogin); err != nil {
			_ = e.store.Set(prev)
			return config.Settings{}, err
		}
	}
	if patch.Enforcement != nil && patch.Enforcement.OverlaySkipAllowed != nil {
		e.session.SetCanSkip(next.Enforcement.OverlaySkipAllowed)
	}

	e.applySchedulePatchLocked(prev, next)

	return next, nil
}

func (e *Engine) GetRuntimeState(now time.Time) config.RuntimeState {
	e.mu.Lock()
	defer e.mu.Unlock()

	settings := e.store.Get()
	return e.runtimeStateLocked(now, settings)
}

func (e *Engine) Pause(mode string, durationSec int, now time.Time) (config.RuntimeState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	switch mode {
	case PauseModeTemporary:
		if durationSec <= 0 {
			return config.RuntimeState{}, errors.New("temporary pause requires positive duration")
		}
		until := now.Add(time.Duration(durationSec) * time.Second)
		e.paused = true
		e.pauseMode = mode
		e.pausedUntil = &until
	case PauseModeIndefinite:
		e.paused = true
		e.pauseMode = mode
		e.pausedUntil = nil
	default:
		return config.RuntimeState{}, errors.New("invalid pause mode")
	}

	settings := e.store.Get()
	return e.runtimeStateLocked(now, settings), nil
}

func (e *Engine) Resume(now time.Time) config.RuntimeState {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.paused = false
	e.pauseMode = ""
	e.pausedUntil = nil
	e.lastTick = now
	e.tickRemainder = 0

	settings := e.store.Get()
	return e.runtimeStateLocked(now, settings)
}

func (e *Engine) SkipCurrentBreak(now time.Time) (config.RuntimeState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	settings := e.store.Get()
	if !settings.Enforcement.OverlaySkipAllowed {
		return config.RuntimeState{}, errors.New("skip is disabled by settings")
	}
	if err := e.session.Skip(); err != nil {
		return config.RuntimeState{}, err
	}
	e.session.ClearIfDone()
	return e.runtimeStateLocked(now, settings), nil
}

func (e *Engine) StartBreakNow(now time.Time) (config.RuntimeState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	settings := e.store.Get()
	if !settings.GlobalEnabled {
		return config.RuntimeState{}, errors.New("global reminders are disabled")
	}
	_ = e.syncPause(now)
	if e.paused {
		// Allow "break now" while paused and treat it as an explicit resume.
		e.paused = false
		e.pauseMode = ""
		e.pausedUntil = nil
	}
	if e.session.IsActive() {
		return config.RuntimeState{}, errors.New("break already active")
	}

	evt := buildImmediateBreakEvent(settings, e.scheduler.NextEyeInSec(settings), e.scheduler.NextStandInSec(settings))
	if evt == nil {
		return config.RuntimeState{}, errors.New("no enabled reminder rules")
	}

	// Manual break should reset cadence to avoid immediate back-to-back reminders.
	e.scheduler.Reset()
	e.lastTick = now
	e.tickRemainder = 0

	e.session.StartBreak(now, evt, settings.Enforcement.OverlaySkipAllowed)
	_ = e.notifier.ShowReminder("Time to rest", buildReasonText(evt.Reasons, evt.BreakSec))
	return e.runtimeStateLocked(now, settings), nil
}

func (e *Engine) runtimeStateLocked(now time.Time, settings config.Settings) config.RuntimeState {
	reasons := nextReasons(e.scheduler.NextEyeInSec(settings), e.scheduler.NextStandInSec(settings))
	return config.RuntimeState{
		Now:                now,
		Paused:             e.paused,
		PauseMode:          e.pauseMode,
		PausedUntil:        e.pausedUntil,
		CurrentSession:     e.session.CurrentView(now),
		NextEyeInSec:       e.scheduler.NextEyeInSec(settings),
		NextStandInSec:     e.scheduler.NextStandInSec(settings),
		NextBreakReason:    reasons,
		GlobalEnabled:      settings.GlobalEnabled,
		TimerMode:          settings.Timer.Mode,
		IdleThresholdSec:   settings.Timer.IdlePauseThresholdSec,
		LastTickActive:     e.lastTickActive,
		CurrentIdleSec:     e.currentIdleSec,
		ShowTrayCountdown:  settings.UI.ShowTrayCountdown,
		OverlayEnabled:     settings.Enforcement.OverlayEnabled,
		OverlaySkipAllowed: settings.Enforcement.OverlaySkipAllowed,
	}
}

func (e *Engine) isTickActive(settings config.Settings) bool {
	if settings.Timer.Mode == config.TimerModeRealTime {
		return true
	}
	return e.currentIdleSec < settings.Timer.IdlePauseThresholdSec
}

func (e *Engine) syncPause(now time.Time) bool {
	if !e.paused {
		return false
	}
	if e.pausedUntil == nil {
		return true
	}
	if now.Before(*e.pausedUntil) {
		return true
	}
	e.paused = false
	e.pauseMode = ""
	e.pausedUntil = nil
	return false
}

func (e *Engine) logTickLocked(now time.Time, settings config.Settings, reason string, rawDeltaSec, appliedDeltaSec int, evt *scheduler.Event) {
	sessionStatus := "none"
	if view := e.session.CurrentView(now); view != nil {
		sessionStatus = view.Status
	}

	evtReasons := ""
	evtBreak := 0
	if evt != nil {
		evtBreak = evt.BreakSec
		reasons := make([]string, 0, len(evt.Reasons))
		for _, r := range evt.Reasons {
			reasons = append(reasons, string(r))
		}
		evtReasons = strings.Join(reasons, "+")
	}

	diag.Logf(
		"engine.tick reason=%s now_unix=%d raw_delta=%d applied_delta=%d idle_sec=%d tick_active=%t paused=%t session=%s next_eye=%d next_stand=%d evt_reasons=%s evt_break=%d",
		reason,
		now.Unix(),
		rawDeltaSec,
		appliedDeltaSec,
		e.currentIdleSec,
		e.lastTickActive,
		e.paused,
		sessionStatus,
		e.scheduler.NextEyeInSec(settings),
		e.scheduler.NextStandInSec(settings),
		evtReasons,
		evtBreak,
	)
}

func (e *Engine) applySchedulePatchLocked(prev, next config.Settings) {
	if prev.GlobalEnabled != next.GlobalEnabled {
		e.scheduler.Reset()
		e.lastTick = time.Time{}
		e.tickRemainder = 0
		return
	}

	if prev.Eye.Enabled != next.Eye.Enabled || prev.Eye.IntervalSec != next.Eye.IntervalSec {
		e.scheduler.ResetEye()
	}
	if prev.Stand.Enabled != next.Stand.Enabled || prev.Stand.IntervalSec != next.Stand.IntervalSec {
		e.scheduler.ResetStand()
	}
}

func buildReasonText(reasons []scheduler.ReminderType, breakSec int) string {
	if len(reasons) == 0 {
		return ""
	}

	label := "Eye"
	if len(reasons) == 2 {
		label = "Eye + Stand"
	} else if reasons[0] == scheduler.ReminderStand {
		label = "Stand"
	}

	return label + " break for " + (time.Duration(breakSec) * time.Second).String()
}

func buildImmediateBreakEvent(settings config.Settings, nextEye, nextStand int) *scheduler.Event {
	reasonKeys := nextReasons(nextEye, nextStand)
	if len(reasonKeys) == 0 {
		return nil
	}

	reasons := make([]scheduler.ReminderType, 0, len(reasonKeys))
	breakSec := 0
	for _, reason := range reasonKeys {
		switch reason {
		case string(scheduler.ReminderEye):
			reasons = append(reasons, scheduler.ReminderEye)
			if settings.Eye.Enabled && settings.Eye.BreakSec > breakSec {
				breakSec = settings.Eye.BreakSec
			}
		case string(scheduler.ReminderStand):
			reasons = append(reasons, scheduler.ReminderStand)
			if settings.Stand.Enabled && settings.Stand.BreakSec > breakSec {
				breakSec = settings.Stand.BreakSec
			}
		}
	}
	if len(reasons) == 0 || breakSec <= 0 {
		return nil
	}
	return &scheduler.Event{
		Reasons:  reasons,
		BreakSec: breakSec,
	}
}

func nextReasons(nextEye, nextStand int) []string {
	reasons := []string{}
	if nextEye < 0 && nextStand < 0 {
		return reasons
	}
	if nextEye >= 0 && nextStand >= 0 {
		if int(math.Abs(float64(nextEye-nextStand))) <= 60 {
			return []string{string(scheduler.ReminderEye), string(scheduler.ReminderStand)}
		}
		if nextEye < nextStand {
			return []string{string(scheduler.ReminderEye)}
		}
		return []string{string(scheduler.ReminderStand)}
	}
	if nextEye >= 0 {
		return []string{string(scheduler.ReminderEye)}
	}
	return []string{string(scheduler.ReminderStand)}
}
