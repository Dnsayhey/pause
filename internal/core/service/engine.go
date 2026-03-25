package service

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"pause/internal/core/reminder"
	"pause/internal/core/scheduler"
	"pause/internal/core/session"
	"pause/internal/core/settings"
	"pause/internal/core/state"
	"pause/internal/logx"
	"pause/internal/platform"
)

type SkipMode string

const (
	SkipModeNormal    SkipMode = "normal"
	SkipModeEmergency SkipMode = "emergency"
)

type BreakHistoryRecorder interface {
	RecordBreak(ctx context.Context, startedAt time.Time, endedAt time.Time, source string, plannedBreakSec int, actualBreakSec int, skipped bool, reminderIDs []int64) error
}

type SettingsStore interface {
	WasCreated() bool
	Get() settings.Settings
	Update(patch settings.SettingsPatch) (settings.Settings, error)
}

type pendingHistoryBreak struct {
	StartedAt       time.Time
	Source          string
	PlannedBreakSec int
	ReminderIDs     []int64
}

type Engine struct {
	mu        sync.Mutex
	startOnce sync.Once

	store     SettingsStore
	reminders []reminder.ReminderConfig
	scheduler *scheduler.Scheduler
	session   *session.Manager
	history   BreakHistoryRecorder

	idleProvider   platform.IdleProvider
	lockProvider   platform.LockStateProvider
	soundPlayer    platform.SoundPlayer
	notifier       platform.Notifier
	startupManager platform.StartupManager

	lastTick      time.Time
	tickRemainder time.Duration

	pausedReminder map[int64]bool

	lastTickActive bool
	currentIdleSec int
	currentLocked  bool

	activeHistoryBreak *pendingHistoryBreak
}

func NewEngine(
	store SettingsStore,
	idleProvider platform.IdleProvider,
	lockProvider platform.LockStateProvider,
	soundPlayer platform.SoundPlayer,
	startupManager platform.StartupManager,
	history BreakHistoryRecorder,
) *Engine {
	if idleProvider == nil {
		idleProvider = platform.NoopIdleProvider{}
	}
	if lockProvider == nil {
		lockProvider = platform.NoopLockStateProvider{}
	}
	if soundPlayer == nil {
		soundPlayer = platform.NoopSoundPlayer{}
	}
	if startupManager == nil {
		startupManager = platform.NoopStartupManager{}
	}

	return &Engine{
		store:          store,
		reminders:      cloneReminderConfigs(nil),
		scheduler:      scheduler.New(),
		session:        session.NewManager(),
		history:        history,
		idleProvider:   idleProvider,
		lockProvider:   lockProvider,
		soundPlayer:    soundPlayer,
		notifier:       platform.NoopNotifier{},
		startupManager: startupManager,
		pausedReminder: map[int64]bool{},
	}
}

func (e *Engine) SetNotifier(notifier platform.Notifier) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if notifier == nil {
		e.notifier = platform.NoopNotifier{}
		return
	}
	e.notifier = notifier
}

func (e *Engine) SyncPlatformSettings() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// First app run after config creation: attempt to enable launch-at-login once.
	if e.store.WasCreated() {
		logx.Infof("startup.auto_launch_at_login attempt=true")
		if err := e.startupManager.SetLaunchAtLogin(true); err != nil {
			logx.Warnf("startup.auto_launch_at_login_err err=%v", err)
			return err
		}
		logx.Infof("startup.auto_launch_at_login enabled=true")
	}
	return nil
}

func (e *Engine) Start(ctx context.Context) {
	e.startOnce.Do(func() {
		logx.Infof("engine.start")
		e.mu.Lock()
		if e.lastTick.IsZero() {
			// Seed the baseline once at startup so the first scheduler tick accounts
			// for the first elapsed second instead of waiting an extra cycle.
			e.lastTick = time.Now()
			e.tickRemainder = 0
		}
		e.mu.Unlock()

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
	locked := e.lockProvider.IsScreenLocked()

	e.mu.Lock()
	defer e.mu.Unlock()

	firstTick := e.lastTick.IsZero()
	if firstTick {
		e.lastTick = now
	}

	settings := e.store.Get()
	effectiveReminders := e.effectiveReminderConfigsLocked(e.reminders)
	wasLocked := e.currentLocked
	e.currentIdleSec = idleSec
	e.currentLocked = locked
	if !wasLocked && locked {
		logx.Infof(
			"engine.screen_locked timer_mode=%s idle_sec=%d threshold_sec=%d",
			settings.Timer.Mode,
			idleSec,
			settings.Timer.IdlePauseThresholdSec,
		)
	} else if wasLocked && !locked {
		logx.Infof(
			"engine.screen_unlocked timer_mode=%s idle_sec=%d threshold_sec=%d",
			settings.Timer.Mode,
			idleSec,
			settings.Timer.IdlePauseThresholdSec,
		)
	}
	e.lastTickActive = e.isTickActive(settings)
	rawDeltaSec := int(now.Sub(e.lastTick).Seconds())
	appliedDeltaSec := rawDeltaSec
	if firstTick {
		e.logTickLocked(now, settings, effectiveReminders, "bootstrap", 0, 0, nil)
		return
	}

	e.session.Tick(now)
	if view := e.session.CurrentView(now); view != nil && view.Status == string(session.StatusCompleted) {
		e.recordBreakCompletedLocked(view)
		logx.Infof(
			"break.completed reasons=%s duration_sec=%d",
			joinReasons(view.Reasons),
			int(view.EndsAt.Sub(view.StartedAt).Seconds()),
		)
		if err := e.soundPlayer.PlayBreakEnd(settings.Sound); err != nil {
			logx.Warnf("break.end_sound_err err=%v", err)
		}
	}
	e.session.ClearIfDone()

	if !settings.GlobalEnabled {
		e.lastTick = now
		e.tickRemainder = 0
		e.logTickLocked(now, settings, effectiveReminders, "global_disabled", rawDeltaSec, appliedDeltaSec, nil)
		return
	}

	if e.session.IsActive() {
		e.lastTick = now
		e.tickRemainder = 0
		e.logTickLocked(now, settings, effectiveReminders, "session_active", rawDeltaSec, appliedDeltaSec, nil)
		return
	}

	if !e.lastTickActive {
		e.lastTick = now
		e.tickRemainder = 0
		e.logTickLocked(now, settings, effectiveReminders, "idle_paused", rawDeltaSec, appliedDeltaSec, nil)
		return
	}

	elapsed := now.Sub(e.lastTick) + e.tickRemainder
	if elapsed < 0 {
		e.lastTick = now
		e.tickRemainder = 0
		e.logTickLocked(now, settings, effectiveReminders, "negative_elapsed", rawDeltaSec, 0, nil)
		return
	}

	rawDeltaSec = int(elapsed / time.Second)
	if rawDeltaSec <= 0 {
		e.lastTick = now
		e.tickRemainder = elapsed
		e.logTickLocked(now, settings, effectiveReminders, "sub_second_elapsed", rawDeltaSec, 0, nil)
		return
	}

	appliedDeltaSec = rawDeltaSec
	e.tickRemainder = elapsed - (time.Duration(appliedDeltaSec) * time.Second)

	evt := e.scheduler.OnActiveSeconds(appliedDeltaSec, effectiveReminders)
	e.lastTick = now
	if evt == nil {
		e.logTickLocked(now, settings, effectiveReminders, "no_event", rawDeltaSec, appliedDeltaSec, nil)
		return
	}

	restEvent, notifyReminderIDs := splitReminderEventByType(evt, effectiveReminders)
	if len(notifyReminderIDs) > 0 {
		e.notifyRemindersLocked(notifyReminderIDs, settings.UI.Language)
	}
	if restEvent == nil {
		e.logTickLocked(now, settings, effectiveReminders, "notification_event", rawDeltaSec, appliedDeltaSec, evt)
		return
	}

	e.session.StartBreak(now, restEvent, settings.Enforcement.OverlaySkipAllowed)
	e.recordBreakStartedLocked(now, "scheduled", restEvent)
	logx.Infof(
		"break.started source=scheduled reasons=%s break_sec=%d skip_allowed=%t",
		joinReminderTypes(restEvent.Reasons),
		restEvent.BreakSec,
		settings.Enforcement.OverlaySkipAllowed,
	)
	e.logTickLocked(now, settings, effectiveReminders, "event", rawDeltaSec, appliedDeltaSec, restEvent)
}

func (e *Engine) GetSettings() settings.Settings {
	return e.store.Get()
}

func (e *Engine) UpdateSettings(patch settings.SettingsPatch) (settings.Settings, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	patchJSON := marshalPatchForLog(patch)
	prev := e.store.Get()
	next, err := e.store.Update(patch)
	if err != nil {
		logx.Warnf("settings.update_err patch=%s err=%v", patchJSON, err)
		return settings.Settings{}, err
	}

	if patch.Enforcement != nil && patch.Enforcement.OverlaySkipAllowed != nil {
		e.session.SetCanSkip(next.Enforcement.OverlaySkipAllowed)
	}

	e.applyGlobalSettingPatchLocked(prev, next)
	logx.Infof("settings.updated patch=%s", patchJSON)

	return next, nil
}

func (e *Engine) SetReminderConfigs(reminders []reminder.ReminderConfig) []reminder.ReminderConfig {
	e.mu.Lock()
	defer e.mu.Unlock()

	next := cloneReminderConfigs(reminders)
	prev := cloneReminderConfigs(e.reminders)
	e.reminders = cloneReminderConfigs(next)
	e.applyReminderConfigPatchLocked(prev, next)
	logx.Infof("reminders.synced count=%d", len(e.reminders))
	return cloneReminderConfigs(e.reminders)
}

func (e *Engine) GetLaunchAtLogin() (bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	enabled, err := e.startupManager.GetLaunchAtLogin()
	if err != nil {
		logx.Warnf("launch_at_login.get_err err=%v", err)
		return false, err
	}
	logx.Debugf("launch_at_login.get enabled=%t", enabled)
	return enabled, nil
}

func (e *Engine) SetLaunchAtLogin(enabled bool) (bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.startupManager.SetLaunchAtLogin(enabled); err != nil {
		logx.Warnf("launch_at_login.set_err requested=%t err=%v", enabled, err)
		return false, err
	}
	actual, err := e.startupManager.GetLaunchAtLogin()
	if err != nil {
		logx.Warnf("launch_at_login.verify_err requested=%t err=%v", enabled, err)
		return false, err
	}
	logx.Infof("launch_at_login.set requested=%t actual=%t", enabled, actual)
	return actual, nil
}

func (e *Engine) GetRuntimeState(now time.Time) state.RuntimeState {
	e.mu.Lock()
	defer e.mu.Unlock()

	settings := e.store.Get()
	return e.runtimeStateLocked(now, settings)
}

func (e *Engine) Pause(now time.Time) (state.RuntimeState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	enabled := false
	prev := e.store.Get()
	next, err := e.store.Update(settings.SettingsPatch{
		GlobalEnabled: &enabled,
	})
	if err != nil {
		return state.RuntimeState{}, err
	}
	e.applyGlobalSettingPatchLocked(prev, next)
	logx.Infof("global_enabled.set enabled=false source=pause")
	return e.runtimeStateLocked(now, next), nil
}

func (e *Engine) Resume(now time.Time) state.RuntimeState {
	e.mu.Lock()
	defer e.mu.Unlock()

	enabled := true
	prev := e.store.Get()
	next, err := e.store.Update(settings.SettingsPatch{
		GlobalEnabled: &enabled,
	})
	if err != nil {
		logx.Warnf("global_enabled.set_err enabled=true err=%v", err)
		return e.runtimeStateLocked(now, prev)
	}
	e.applyGlobalSettingPatchLocked(prev, next)
	logx.Infof("global_enabled.set enabled=true source=resume")
	return e.runtimeStateLocked(now, next)
}

func (e *Engine) PauseReminder(reminderID int64, now time.Time) (state.RuntimeState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	key := normalizeReminderID(reminderID)
	if key <= 0 {
		return state.RuntimeState{}, errors.New("invalid reminder reason")
	}
	if _, ok := findReminderByID(e.reminders, key); !ok {
		return state.RuntimeState{}, errors.New("unknown reminder reason")
	}
	wasPaused := e.pausedReminder[key]
	e.pausedReminder[key] = true
	logx.Infof("reminder.pause reason=%d already_paused=%t", key, wasPaused)

	settings := e.store.Get()
	return e.runtimeStateLocked(now, settings), nil
}

func (e *Engine) ResumeReminder(reminderID int64, now time.Time) (state.RuntimeState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	key := normalizeReminderID(reminderID)
	if key <= 0 {
		return state.RuntimeState{}, errors.New("invalid reminder reason")
	}
	wasPaused := e.pausedReminder[key]
	delete(e.pausedReminder, key)
	logx.Infof("reminder.resume reason=%d was_paused=%t", key, wasPaused)

	settings := e.store.Get()
	return e.runtimeStateLocked(now, settings), nil
}

func (e *Engine) SkipCurrentBreak(now time.Time, mode SkipMode) (state.RuntimeState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	settings := e.store.Get()
	view := e.session.CurrentView(now)
	switch mode {
	case "", SkipModeNormal:
		if !settings.Enforcement.OverlaySkipAllowed {
			return state.RuntimeState{}, errors.New("skip is disabled by settings")
		}
	case SkipModeEmergency:
		// Emergency path: allow explicit user escape from enforced overlay.
		e.session.SetCanSkip(true)
	default:
		return state.RuntimeState{}, errors.New("invalid skip mode")
	}

	if err := e.session.Skip(); err != nil {
		logx.Warnf("break.skip_err mode=%s err=%v", mode, err)
		return state.RuntimeState{}, err
	}
	if view != nil {
		e.recordBreakSkippedLocked(now, view)
		logx.Infof(
			"break.skipped mode=%s reasons=%s remaining_sec=%d",
			mode,
			joinReasons(view.Reasons),
			view.RemainingSec,
		)
	}
	e.session.ClearIfDone()
	return e.runtimeStateLocked(now, settings), nil
}

func (e *Engine) StartBreakNow(now time.Time) (state.RuntimeState, error) {
	return e.StartBreakNowForReason(0, now)
}

func (e *Engine) StartBreakNowForReason(reason int64, now time.Time) (state.RuntimeState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	settings := e.store.Get()
	if !settings.GlobalEnabled {
		return state.RuntimeState{}, errors.New("global reminders are disabled")
	}
	if e.session.IsActive() {
		return state.RuntimeState{}, errors.New("break already active")
	}

	effectiveReminders := e.effectiveReminderConfigsLocked(e.reminders)
	evt := buildImmediateBreakEvent(effectiveReminders, e.scheduler.NextByID(effectiveReminders), reason)
	if evt == nil {
		return state.RuntimeState{}, errors.New("no enabled reminder rules")
	}

	// Manual break should reset cadence for selected reminder reasons to avoid
	// immediate back-to-back reminders, without affecting unrelated reminders.
	resetSchedulerByReasons(e.scheduler, evt.Reasons)
	e.lastTick = now
	e.tickRemainder = 0

	e.session.StartBreak(now, evt, settings.Enforcement.OverlaySkipAllowed)
	e.recordBreakStartedLocked(now, "manual", evt)
	logx.Infof(
		"break.started source=manual reasons=%s break_sec=%d skip_allowed=%t forced_reason=%d",
		joinReminderTypes(evt.Reasons),
		evt.BreakSec,
		settings.Enforcement.OverlaySkipAllowed,
		normalizeReminderID(reason),
	)
	return e.runtimeStateLocked(now, settings), nil
}

func (e *Engine) runtimeStateLocked(now time.Time, settings settings.Settings) state.RuntimeState {
	effectiveReminders := e.effectiveReminderConfigsLocked(e.reminders)
	reminders := make([]state.ReminderRuntime, 0, len(e.reminders))
	for _, reminder := range e.reminders {
		paused := e.pausedReminder[reminder.ID]
		nextIn := e.scheduler.NextInSec(effectiveReminders, reminder.ID)
		reminders = append(reminders, state.ReminderRuntime{
			ID:           reminder.ID,
			Name:         reminder.Name,
			ReminderType: reminder.ReminderType,
			Enabled:      reminder.Enabled,
			Paused:       paused,
			NextInSec:    nextIn,
			IntervalSec:  reminder.IntervalSec,
			BreakSec:     reminder.BreakSec,
		})
	}
	reasons := nextReasons(reminders, e.reminders)
	return state.RuntimeState{
		Now:                now,
		CurrentSession:     e.session.CurrentView(now),
		Reminders:          reminders,
		NextBreakReason:    reasons,
		GlobalEnabled:      settings.GlobalEnabled,
		TimerMode:          settings.Timer.Mode,
		IdleThresholdSec:   settings.Timer.IdlePauseThresholdSec,
		LastTickActive:     e.lastTickActive,
		CurrentIdleSec:     e.currentIdleSec,
		ShowTrayCountdown:  settings.UI.ShowTrayCountdown,
		OverlaySkipAllowed: settings.Enforcement.OverlaySkipAllowed,
	}
}

func (e *Engine) isTickActive(cfg settings.Settings) bool {
	if cfg.Timer.Mode == settings.TimerModeRealTime {
		return true
	}
	if e.currentLocked {
		return false
	}
	return e.currentIdleSec < cfg.Timer.IdlePauseThresholdSec
}

func reminderIDsFromEvent(evt *scheduler.Event) []int64 {
	if evt == nil || len(evt.Reasons) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(evt.Reasons))
	for _, reason := range evt.Reasons {
		id := normalizeReminderID(int64(reason))
		if id <= 0 {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	uniq := ids[:0]
	var last int64
	lastSet := false
	for _, id := range ids {
		if lastSet && id == last {
			continue
		}
		uniq = append(uniq, id)
		last = id
		lastSet = true
	}
	return uniq
}

func (e *Engine) recordBreakStartedLocked(now time.Time, source string, evt *scheduler.Event) {
	e.activeHistoryBreak = nil
	if e.history == nil || evt == nil || evt.BreakSec <= 0 {
		return
	}

	e.activeHistoryBreak = &pendingHistoryBreak{
		StartedAt:       now,
		Source:          source,
		PlannedBreakSec: evt.BreakSec,
		ReminderIDs:     reminderIDsFromEvent(evt),
	}
}

func (e *Engine) recordBreakCompletedLocked(view *state.BreakSessionView) {
	if e.history == nil || view == nil || e.activeHistoryBreak == nil {
		e.activeHistoryBreak = nil
		return
	}
	actualBreakSec := int(view.EndsAt.Sub(view.StartedAt).Seconds())
	if actualBreakSec < 0 {
		actualBreakSec = 0
	}
	record := *e.activeHistoryBreak
	if err := e.history.RecordBreak(context.Background(), record.StartedAt, view.EndsAt, record.Source, record.PlannedBreakSec, actualBreakSec, false, record.ReminderIDs); err != nil {
		logx.Warnf("history.break_complete_err source=%s err=%v", record.Source, err)
	}
	e.activeHistoryBreak = nil
}

func (e *Engine) recordBreakSkippedLocked(now time.Time, view *state.BreakSessionView) {
	if e.history == nil || view == nil || e.activeHistoryBreak == nil {
		e.activeHistoryBreak = nil
		return
	}
	actualBreakSec := int(now.Sub(view.StartedAt).Seconds())
	if actualBreakSec < 0 {
		actualBreakSec = 0
	}
	record := *e.activeHistoryBreak
	if err := e.history.RecordBreak(context.Background(), record.StartedAt, now, record.Source, record.PlannedBreakSec, actualBreakSec, true, record.ReminderIDs); err != nil {
		logx.Warnf("history.break_skip_err source=%s err=%v", record.Source, err)
	}
	e.activeHistoryBreak = nil
}

func (e *Engine) logTickLocked(now time.Time, settings settings.Settings, reminders []reminder.ReminderConfig, reason string, rawDeltaSec, appliedDeltaSec int, evt *scheduler.Event) {
	sessionStatus := "none"
	if view := e.session.CurrentView(now); view != nil {
		sessionStatus = view.Status
	}
	nextByID := e.scheduler.NextByID(reminders)
	nextSummary := "none"
	if len(nextByID) > 0 {
		parts := make([]string, 0, len(nextByID))
		for id, next := range nextByID {
			parts = append(parts, strconv.FormatInt(id, 10)+"="+strconv.Itoa(next))
		}
		sort.Strings(parts)
		nextSummary = strings.Join(parts, ",")
	}

	evtReasons := ""
	evtBreak := 0
	if evt != nil {
		evtBreak = evt.BreakSec
		reasons := make([]string, 0, len(evt.Reasons))
		for _, r := range evt.Reasons {
			reasons = append(reasons, strconv.FormatInt(int64(r), 10))
		}
		evtReasons = strings.Join(reasons, "+")
	}

	logx.Debugf(
		"engine.tick reason=%s now_unix=%d raw_delta=%d applied_delta=%d idle_sec=%d tick_active=%t session=%s next=%s evt_reasons=%s evt_break=%d",
		reason,
		now.Unix(),
		rawDeltaSec,
		appliedDeltaSec,
		e.currentIdleSec,
		e.lastTickActive,
		sessionStatus,
		nextSummary,
		evtReasons,
		evtBreak,
	)
}

func (e *Engine) applyGlobalSettingPatchLocked(prev, next settings.Settings) {
	if prev.GlobalEnabled != next.GlobalEnabled {
		e.scheduler.Reset()
		e.pausedReminder = map[int64]bool{}
		e.lastTick = time.Time{}
		e.tickRemainder = 0
	}
}

func (e *Engine) applyReminderConfigPatchLocked(prev, next []reminder.ReminderConfig) {
	prevByID := map[int64]reminder.ReminderConfig{}
	for _, reminder := range prev {
		prevByID[reminder.ID] = reminder
	}
	nextByID := map[int64]reminder.ReminderConfig{}
	for _, reminder := range next {
		nextByID[reminder.ID] = reminder
	}

	ids := map[int64]struct{}{}
	for id := range prevByID {
		ids[id] = struct{}{}
	}
	for id := range nextByID {
		ids[id] = struct{}{}
	}

	for id := range ids {
		p, hasPrev := prevByID[id]
		n, hasNext := nextByID[id]
		changed := !hasPrev || !hasNext ||
			p.Enabled != n.Enabled ||
			p.IntervalSec != n.IntervalSec
		if !changed {
			continue
		}
		e.scheduler.ResetByID(id)
		delete(e.pausedReminder, id)
	}
}

func (e *Engine) effectiveReminderConfigsLocked(reminders []reminder.ReminderConfig) []reminder.ReminderConfig {
	updated := make([]reminder.ReminderConfig, 0, len(reminders))
	for _, reminder := range reminders {
		next := reminder
		if next.Enabled && e.pausedReminder[next.ID] {
			next.Enabled = false
		}
		updated = append(updated, next)
	}
	return updated
}

func buildImmediateBreakEvent(reminders []reminder.ReminderConfig, nextByID map[int64]int, forcedReason int64) *scheduler.Event {
	reasonKey := normalizeReminderID(forcedReason)
	if reasonKey <= 0 {
		reasonKey = selectImmediateReason(nextByID)
	}
	if reasonKey <= 0 {
		return nil
	}
	cfg, ok := findReminderByID(reminders, reasonKey)
	if !ok || !cfg.Enabled || cfg.BreakSec <= 0 || !isRestReminderType(cfg.ReminderType) {
		return nil
	}
	return &scheduler.Event{
		Reasons:  []scheduler.ReminderType{scheduler.ReminderType(reasonKey)},
		BreakSec: cfg.BreakSec,
	}
}

func normalizeReminderID(id int64) int64 {
	if id <= 0 {
		return 0
	}
	return id
}

func cloneReminderConfigs(reminders []reminder.ReminderConfig) []reminder.ReminderConfig {
	if len(reminders) == 0 {
		return nil
	}
	cloned := make([]reminder.ReminderConfig, 0, len(reminders))
	cloned = append(cloned, reminders...)
	return cloned
}

func findReminderByID(reminders []reminder.ReminderConfig, id int64) (reminder.ReminderConfig, bool) {
	norm := normalizeReminderID(id)
	for _, cfg := range reminders {
		if cfg.ID == norm {
			return cfg, true
		}
	}
	return reminder.ReminderConfig{}, false
}

func selectImmediateReason(nextByID map[int64]int) int64 {
	var bestID int64
	bestNext := -1
	for id, next := range nextByID {
		if next < 0 {
			continue
		}
		if bestNext < 0 || next < bestNext || (next == bestNext && (bestID == 0 || id < bestID)) {
			bestID = id
			bestNext = next
		}
	}
	return bestID
}

func resetSchedulerByReasons(s *scheduler.Scheduler, reasons []scheduler.ReminderType) {
	if s == nil || len(reasons) == 0 {
		return
	}

	seen := map[scheduler.ReminderType]struct{}{}
	for _, reason := range reasons {
		if _, ok := seen[reason]; ok {
			continue
		}
		seen[reason] = struct{}{}
		s.ResetByID(int64(reason))
	}
}

func joinReminderTypes(reasons []scheduler.ReminderType) string {
	if len(reasons) == 0 {
		return "none"
	}

	labels := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		labels = append(labels, strconv.FormatInt(int64(reason), 10))
	}
	return strings.Join(labels, "+")
}

func joinReasons(reasons []int64) string {
	if len(reasons) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		parts = append(parts, strconv.FormatInt(reason, 10))
	}
	return strings.Join(parts, "+")
}

func marshalPatchForLog(patch settings.SettingsPatch) string {
	raw, err := json.Marshal(patch)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func nextReasons(reminders []state.ReminderRuntime, defs []reminder.ReminderConfig) []int64 {
	restReminderIDs := map[int64]struct{}{}
	for _, def := range defs {
		if !isRestReminderType(def.ReminderType) {
			continue
		}
		restReminderIDs[def.ID] = struct{}{}
	}

	minNext := -1
	for _, reminder := range reminders {
		if _, isRest := restReminderIDs[reminder.ID]; !isRest {
			continue
		}
		if !reminder.Enabled || reminder.Paused || reminder.NextInSec < 0 {
			continue
		}
		if minNext < 0 || reminder.NextInSec < minNext {
			minNext = reminder.NextInSec
		}
	}
	if minNext < 0 {
		return []int64{}
	}

	reasons := make([]int64, 0, len(reminders))
	for _, reminder := range reminders {
		if _, isRest := restReminderIDs[reminder.ID]; !isRest {
			continue
		}
		if !reminder.Enabled || reminder.Paused || reminder.NextInSec < 0 {
			continue
		}
		if reminder.NextInSec-minNext <= 60 {
			reasons = append(reasons, reminder.ID)
		}
	}
	sort.Slice(reasons, func(i, j int) bool { return reasons[i] < reasons[j] })
	return reasons
}

func isRestReminderType(reminderType string) bool {
	return strings.ToLower(strings.TrimSpace(reminderType)) != "notify"
}

func splitReminderEventByType(evt *scheduler.Event, reminders []reminder.ReminderConfig) (*scheduler.Event, []int64) {
	if evt == nil || len(evt.Reasons) == 0 {
		return nil, nil
	}

	byID := make(map[int64]reminder.ReminderConfig, len(reminders))
	for _, reminder := range reminders {
		byID[reminder.ID] = reminder
	}

	restReasons := make([]scheduler.ReminderType, 0, len(evt.Reasons))
	notifyReminderIDs := make([]int64, 0, len(evt.Reasons))
	restBreakSec := 0

	for _, reason := range evt.Reasons {
		id := normalizeReminderID(int64(reason))
		reminder, ok := byID[id]
		if !ok {
			restReasons = append(restReasons, reason)
			if evt.BreakSec > restBreakSec {
				restBreakSec = evt.BreakSec
			}
			continue
		}
		if isRestReminderType(reminder.ReminderType) {
			restReasons = append(restReasons, reason)
			if reminder.BreakSec > restBreakSec {
				restBreakSec = reminder.BreakSec
			}
			continue
		}
		notifyReminderIDs = append(notifyReminderIDs, id)
	}

	sort.Slice(notifyReminderIDs, func(i, j int) bool { return notifyReminderIDs[i] < notifyReminderIDs[j] })
	if len(notifyReminderIDs) > 1 {
		uniq := notifyReminderIDs[:0]
		var last int64
		lastSet := false
		for _, id := range notifyReminderIDs {
			if lastSet && id == last {
				continue
			}
			uniq = append(uniq, id)
			last = id
			lastSet = true
		}
		notifyReminderIDs = uniq
	}

	if len(restReasons) == 0 {
		return nil, notifyReminderIDs
	}
	if restBreakSec <= 0 {
		restBreakSec = evt.BreakSec
		if restBreakSec <= 0 {
			restBreakSec = 1
		}
	}

	return &scheduler.Event{
		Reasons:  restReasons,
		BreakSec: restBreakSec,
	}, notifyReminderIDs
}

func (e *Engine) notifyRemindersLocked(reminderIDs []int64, language string) {
	if len(reminderIDs) == 0 || e.notifier == nil {
		return
	}
	names := make([]string, 0, len(reminderIDs))
	byID := make(map[int64]reminder.ReminderConfig, len(e.reminders))
	for _, reminder := range e.reminders {
		byID[reminder.ID] = reminder
	}
	for _, id := range reminderIDs {
		reminder, ok := byID[id]
		if !ok {
			continue
		}
		name := strings.TrimSpace(reminder.Name)
		if name == "" {
			name = strconv.FormatInt(reminder.ID, 10)
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return
	}

	title := "Reminder"
	body := strings.Join(names, " · ")
	if language == settings.UILanguageZhCN {
		title = "提醒"
	}
	notifier := e.notifier
	keyParts := make([]string, 0, len(reminderIDs))
	for _, id := range reminderIDs {
		keyParts = append(keyParts, strconv.FormatInt(id, 10))
	}
	reminderKey := strings.Join(keyParts, "+")
	go func(n platform.Notifier, t string, b string, key string) {
		if err := n.ShowReminder(t, b); err != nil {
			logx.Warnf("reminder.notification_err reminders=%s err=%v", key, err)
			return
		}
		logx.Infof("reminder.notification_sent reminders=%s", key)
	}(notifier, title, body, reminderKey)
}
