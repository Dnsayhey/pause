package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"pause/internal/core/config"
	"pause/internal/core/scheduler"
	"pause/internal/core/session"
	"pause/internal/logx"
	"pause/internal/platform"
)

type SkipMode string

const (
	SkipModeNormal    SkipMode = "normal"
	SkipModeEmergency SkipMode = "emergency"
)

type BreakHistoryRecorder interface {
	StartBreak(sessionID string, startedAt time.Time, source string, plannedBreakSec int, reminderIDs []string) error
	CompleteBreak(sessionID string, endedAt time.Time, actualBreakSec int) error
	SkipBreak(sessionID string, skippedAt time.Time, actualBreakSec int) error
}

type Engine struct {
	mu        sync.Mutex
	startOnce sync.Once

	store     *config.Store
	reminders []config.ReminderConfig
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

	pausedReminder map[string]bool

	lastTickActive bool
	currentIdleSec int
	currentLocked  bool

	activeHistorySessionID string
}

func NewEngine(
	store *config.Store,
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
		reminders:      config.NormalizeReminderConfigs(nil),
		scheduler:      scheduler.New(),
		session:        session.NewManager(),
		history:        history,
		idleProvider:   idleProvider,
		lockProvider:   lockProvider,
		soundPlayer:    soundPlayer,
		notifier:       platform.NoopNotifier{},
		startupManager: startupManager,
		pausedReminder: map[string]bool{},
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

func (e *Engine) GetSettings() config.Settings {
	return e.store.Get()
}

func (e *Engine) UpdateSettings(patch config.SettingsPatch) (config.Settings, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	patchJSON := marshalPatchForLog(patch)
	prev := e.store.Get()
	next, err := e.store.Update(patch)
	if err != nil {
		logx.Warnf("settings.update_err patch=%s err=%v", patchJSON, err)
		return config.Settings{}, err
	}

	if patch.Enforcement != nil && patch.Enforcement.OverlaySkipAllowed != nil {
		e.session.SetCanSkip(next.Enforcement.OverlaySkipAllowed)
	}

	e.applyGlobalSettingPatchLocked(prev, next)
	logx.Infof("settings.updated patch=%s", patchJSON)

	return next, nil
}

func (e *Engine) SetReminderConfigs(reminders []config.ReminderConfig) []config.ReminderConfig {
	e.mu.Lock()
	defer e.mu.Unlock()

	next := config.NormalizeReminderConfigsKeepEmpty(reminders)
	prev := cloneReminderConfigs(e.reminders)
	e.reminders = cloneReminderConfigs(next)
	e.applyReminderConfigPatchLocked(prev, next)
	logx.Infof("reminders.synced count=%d", len(e.reminders))
	return cloneReminderConfigs(e.reminders)
}

func (e *Engine) UpdateReminderConfigs(patches []config.ReminderPatch) ([]config.ReminderConfig, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if err := validateReminderPatchesForUpdate(e.reminders, patches); err != nil {
		return nil, err
	}

	next := config.ApplyReminderPatches(e.reminders, patches)
	prev := cloneReminderConfigs(e.reminders)
	e.reminders = cloneReminderConfigs(next)
	e.applyReminderConfigPatchLocked(prev, next)
	logx.Infof("reminders.updated patches=%s count=%d", marshalReminderPatchesForLog(patches), len(e.reminders))
	return cloneReminderConfigs(e.reminders), nil
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

func (e *Engine) GetRuntimeState(now time.Time) config.RuntimeState {
	e.mu.Lock()
	defer e.mu.Unlock()

	settings := e.store.Get()
	return e.runtimeStateLocked(now, settings)
}

func (e *Engine) Pause(now time.Time) (config.RuntimeState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	enabled := false
	prev := e.store.Get()
	next, err := e.store.Update(config.SettingsPatch{
		GlobalEnabled: &enabled,
	})
	if err != nil {
		return config.RuntimeState{}, err
	}
	e.applyGlobalSettingPatchLocked(prev, next)
	logx.Infof("global_enabled.set enabled=false source=pause")
	return e.runtimeStateLocked(now, next), nil
}

func (e *Engine) Resume(now time.Time) config.RuntimeState {
	e.mu.Lock()
	defer e.mu.Unlock()

	enabled := true
	prev := e.store.Get()
	next, err := e.store.Update(config.SettingsPatch{
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

func (e *Engine) PauseReminder(reason string, now time.Time) (config.RuntimeState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	key := normalizeReminderID(reason)
	if key == "" {
		return config.RuntimeState{}, errors.New("invalid reminder reason")
	}
	if _, ok := config.ReminderByID(e.reminders, key); !ok {
		return config.RuntimeState{}, errors.New("unknown reminder reason")
	}
	wasPaused := e.pausedReminder[key]
	e.pausedReminder[key] = true
	logx.Infof("reminder.pause reason=%s already_paused=%t", key, wasPaused)

	settings := e.store.Get()
	return e.runtimeStateLocked(now, settings), nil
}

func (e *Engine) ResumeReminder(reason string, now time.Time) (config.RuntimeState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	key := normalizeReminderID(reason)
	if key == "" {
		return config.RuntimeState{}, errors.New("invalid reminder reason")
	}
	wasPaused := e.pausedReminder[key]
	delete(e.pausedReminder, key)
	logx.Infof("reminder.resume reason=%s was_paused=%t", key, wasPaused)

	settings := e.store.Get()
	return e.runtimeStateLocked(now, settings), nil
}

func (e *Engine) SkipCurrentBreak(now time.Time, mode SkipMode) (config.RuntimeState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	settings := e.store.Get()
	view := e.session.CurrentView(now)
	switch mode {
	case "", SkipModeNormal:
		if !settings.Enforcement.OverlaySkipAllowed {
			return config.RuntimeState{}, errors.New("skip is disabled by settings")
		}
	case SkipModeEmergency:
		// Emergency path: allow explicit user escape from enforced overlay.
		e.session.SetCanSkip(true)
	default:
		return config.RuntimeState{}, errors.New("invalid skip mode")
	}

	if err := e.session.Skip(); err != nil {
		logx.Warnf("break.skip_err mode=%s err=%v", mode, err)
		return config.RuntimeState{}, err
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

func (e *Engine) StartBreakNow(now time.Time) (config.RuntimeState, error) {
	return e.StartBreakNowForReason("", now)
}

func (e *Engine) StartBreakNowForReason(reason string, now time.Time) (config.RuntimeState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	settings := e.store.Get()
	if !settings.GlobalEnabled {
		return config.RuntimeState{}, errors.New("global reminders are disabled")
	}
	if e.session.IsActive() {
		return config.RuntimeState{}, errors.New("break already active")
	}

	effectiveReminders := e.effectiveReminderConfigsLocked(e.reminders)
	evt := buildImmediateBreakEvent(effectiveReminders, e.scheduler.NextByID(effectiveReminders), reason)
	if evt == nil {
		return config.RuntimeState{}, errors.New("no enabled reminder rules")
	}

	// Manual break should reset cadence for selected reminder reasons to avoid
	// immediate back-to-back reminders, without affecting unrelated reminders.
	resetSchedulerByReasons(e.scheduler, evt.Reasons)
	e.lastTick = now
	e.tickRemainder = 0

	e.session.StartBreak(now, evt, settings.Enforcement.OverlaySkipAllowed)
	e.recordBreakStartedLocked(now, "manual", evt)
	logx.Infof(
		"break.started source=manual reasons=%s break_sec=%d skip_allowed=%t forced_reason=%s",
		joinReminderTypes(evt.Reasons),
		evt.BreakSec,
		settings.Enforcement.OverlaySkipAllowed,
		normalizeReminderID(reason),
	)
	return e.runtimeStateLocked(now, settings), nil
}

func (e *Engine) runtimeStateLocked(now time.Time, settings config.Settings) config.RuntimeState {
	effectiveReminders := e.effectiveReminderConfigsLocked(e.reminders)
	reminders := make([]config.ReminderRuntime, 0, len(e.reminders))
	for _, reminder := range e.reminders {
		paused := e.pausedReminder[reminder.ID]
		nextIn := e.scheduler.NextInSec(effectiveReminders, reminder.ID)
		reminders = append(reminders, config.ReminderRuntime{
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
	return config.RuntimeState{
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

func (e *Engine) isTickActive(settings config.Settings) bool {
	if settings.Timer.Mode == config.TimerModeRealTime {
		return true
	}
	if e.currentLocked {
		return false
	}
	return e.currentIdleSec < settings.Timer.IdlePauseThresholdSec
}

func newHistorySessionID() string {
	return uuid.NewString()
}

func reminderIDsFromEvent(evt *scheduler.Event) []string {
	if evt == nil || len(evt.Reasons) == 0 {
		return nil
	}
	ids := make([]string, 0, len(evt.Reasons))
	for _, reason := range evt.Reasons {
		id := normalizeReminderID(string(reason))
		if id == "" {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}
	sort.Strings(ids)
	uniq := ids[:0]
	last := ""
	for _, id := range ids {
		if id == last {
			continue
		}
		uniq = append(uniq, id)
		last = id
	}
	return uniq
}

func (e *Engine) recordBreakStartedLocked(now time.Time, source string, evt *scheduler.Event) {
	e.activeHistorySessionID = ""
	if e.history == nil || evt == nil || evt.BreakSec <= 0 {
		return
	}

	sessionID := newHistorySessionID()
	if err := e.history.StartBreak(sessionID, now, source, evt.BreakSec, reminderIDsFromEvent(evt)); err != nil {
		logx.Warnf("history.break_start_err session_id=%s source=%s err=%v", sessionID, source, err)
		return
	}
	e.activeHistorySessionID = sessionID
}

func (e *Engine) recordBreakCompletedLocked(view *config.BreakSessionView) {
	if e.history == nil || view == nil || e.activeHistorySessionID == "" {
		e.activeHistorySessionID = ""
		return
	}
	actualBreakSec := int(view.EndsAt.Sub(view.StartedAt).Seconds())
	if actualBreakSec < 0 {
		actualBreakSec = 0
	}
	if err := e.history.CompleteBreak(e.activeHistorySessionID, view.EndsAt, actualBreakSec); err != nil {
		logx.Warnf("history.break_complete_err session_id=%s err=%v", e.activeHistorySessionID, err)
	}
	e.activeHistorySessionID = ""
}

func (e *Engine) recordBreakSkippedLocked(now time.Time, view *config.BreakSessionView) {
	if e.history == nil || view == nil || e.activeHistorySessionID == "" {
		e.activeHistorySessionID = ""
		return
	}
	actualBreakSec := int(now.Sub(view.StartedAt).Seconds())
	if actualBreakSec < 0 {
		actualBreakSec = 0
	}
	if err := e.history.SkipBreak(e.activeHistorySessionID, now, actualBreakSec); err != nil {
		logx.Warnf("history.break_skip_err session_id=%s err=%v", e.activeHistorySessionID, err)
	}
	e.activeHistorySessionID = ""
}

func (e *Engine) logTickLocked(now time.Time, settings config.Settings, reminders []config.ReminderConfig, reason string, rawDeltaSec, appliedDeltaSec int, evt *scheduler.Event) {
	sessionStatus := "none"
	if view := e.session.CurrentView(now); view != nil {
		sessionStatus = view.Status
	}
	nextByID := e.scheduler.NextByID(reminders)
	nextSummary := "none"
	if len(nextByID) > 0 {
		parts := make([]string, 0, len(nextByID))
		for id, next := range nextByID {
			parts = append(parts, id+"="+strconv.Itoa(next))
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
			reasons = append(reasons, string(r))
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

func (e *Engine) applyGlobalSettingPatchLocked(prev, next config.Settings) {
	if prev.GlobalEnabled != next.GlobalEnabled {
		e.scheduler.Reset()
		e.pausedReminder = map[string]bool{}
		e.lastTick = time.Time{}
		e.tickRemainder = 0
	}
}

func (e *Engine) applyReminderConfigPatchLocked(prev, next []config.ReminderConfig) {
	prevByID := map[string]config.ReminderConfig{}
	for _, reminder := range prev {
		prevByID[reminder.ID] = reminder
	}
	nextByID := map[string]config.ReminderConfig{}
	for _, reminder := range next {
		nextByID[reminder.ID] = reminder
	}

	ids := map[string]struct{}{}
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

func (e *Engine) effectiveReminderConfigsLocked(reminders []config.ReminderConfig) []config.ReminderConfig {
	updated := make([]config.ReminderConfig, 0, len(reminders))
	for _, reminder := range reminders {
		next := reminder
		if next.Enabled && e.pausedReminder[next.ID] {
			next.Enabled = false
		}
		updated = append(updated, next)
	}
	return updated
}

func buildImmediateBreakEvent(reminders []config.ReminderConfig, nextByID map[string]int, forcedReason string) *scheduler.Event {
	reasonKey := normalizeReminderID(forcedReason)
	if reasonKey == "" {
		reasonKey = selectImmediateReason(nextByID)
	}
	if reasonKey == "" {
		return nil
	}
	reminder, ok := config.ReminderByID(reminders, reasonKey)
	if !ok || !reminder.Enabled || reminder.BreakSec <= 0 || !isRestReminderType(reminder.ReminderType) {
		return nil
	}
	return &scheduler.Event{
		Reasons:  []scheduler.ReminderType{scheduler.ReminderType(reasonKey)},
		BreakSec: reminder.BreakSec,
	}
}

func normalizeReminderID(reason string) string {
	return config.NormalizeReminderID(reason)
}

func selectImmediateReason(nextByID map[string]int) string {
	bestID := ""
	bestNext := -1
	for id, next := range nextByID {
		if next < 0 {
			continue
		}
		if bestNext < 0 || next < bestNext || (next == bestNext && id < bestID) {
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
		s.ResetByID(string(reason))
	}
}

func joinReminderTypes(reasons []scheduler.ReminderType) string {
	if len(reasons) == 0 {
		return "none"
	}

	labels := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		labels = append(labels, string(reason))
	}
	return strings.Join(labels, "+")
}

func joinReasons(reasons []string) string {
	if len(reasons) == 0 {
		return "none"
	}
	return strings.Join(reasons, "+")
}

func marshalPatchForLog(patch config.SettingsPatch) string {
	raw, err := json.Marshal(patch)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func marshalReminderPatchesForLog(patches []config.ReminderPatch) string {
	raw, err := json.Marshal(patches)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func validateReminderPatchesForUpdate(reminders []config.ReminderConfig, patches []config.ReminderPatch) error {
	if len(patches) == 0 {
		return nil
	}

	knownIDs := make(map[string]struct{}, len(reminders))
	for _, reminder := range reminders {
		id := config.NormalizeReminderID(reminder.ID)
		if id == "" {
			continue
		}
		knownIDs[id] = struct{}{}
	}

	for _, patch := range patches {
		id := config.NormalizeReminderID(patch.ID)
		if id == "" {
			return errors.New("reminder id is required")
		}
		if _, ok := knownIDs[id]; !ok {
			return fmt.Errorf("reminder id %q not found", id)
		}
		if patch.Name != nil && strings.TrimSpace(*patch.Name) == "" {
			return errors.New("reminder name is required")
		}
		if patch.IntervalSec != nil && *patch.IntervalSec <= 0 {
			return errors.New("reminder intervalSec must be > 0")
		}
		if patch.BreakSec != nil && *patch.BreakSec <= 0 {
			return errors.New("reminder breakSec must be > 0")
		}
		if patch.ReminderType != nil {
			switch strings.ToLower(strings.TrimSpace(*patch.ReminderType)) {
			case "rest", "notify":
			default:
				return errors.New("reminder reminderType must be rest or notify")
			}
		}
	}
	return nil
}

func nextReasons(reminders []config.ReminderRuntime, defs []config.ReminderConfig) []string {
	restReminderIDs := map[string]struct{}{}
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
		return []string{}
	}

	reasons := make([]string, 0, len(reminders))
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
	sort.Strings(reasons)
	return reasons
}

func isRestReminderType(reminderType string) bool {
	return strings.ToLower(strings.TrimSpace(reminderType)) != "notify"
}

func splitReminderEventByType(evt *scheduler.Event, reminders []config.ReminderConfig) (*scheduler.Event, []string) {
	if evt == nil || len(evt.Reasons) == 0 {
		return nil, nil
	}

	byID := make(map[string]config.ReminderConfig, len(reminders))
	for _, reminder := range reminders {
		byID[reminder.ID] = reminder
	}

	restReasons := make([]scheduler.ReminderType, 0, len(evt.Reasons))
	notifyReminderIDs := make([]string, 0, len(evt.Reasons))
	restBreakSec := 0

	for _, reason := range evt.Reasons {
		id := normalizeReminderID(string(reason))
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

	sort.Strings(notifyReminderIDs)
	if len(notifyReminderIDs) > 1 {
		uniq := notifyReminderIDs[:0]
		last := ""
		for _, id := range notifyReminderIDs {
			if id == last {
				continue
			}
			uniq = append(uniq, id)
			last = id
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

func (e *Engine) notifyRemindersLocked(reminderIDs []string, language string) {
	if len(reminderIDs) == 0 || e.notifier == nil {
		return
	}
	names := make([]string, 0, len(reminderIDs))
	byID := make(map[string]config.ReminderConfig, len(e.reminders))
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
			name = reminder.ID
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return
	}

	title := "Reminder"
	body := strings.Join(names, " · ")
	if language == config.UILanguageZhCN {
		title = "提醒"
	}
	notifier := e.notifier
	reminderKey := strings.Join(reminderIDs, "+")
	go func(n platform.Notifier, t string, b string, key string) {
		if err := n.ShowReminder(t, b); err != nil {
			logx.Warnf("reminder.notification_err reminders=%s err=%v", key, err)
			return
		}
		logx.Infof("reminder.notification_sent reminders=%s", key)
	}(notifier, title, body, reminderKey)
}

func cloneReminderConfigs(reminders []config.ReminderConfig) []config.ReminderConfig {
	if len(reminders) == 0 {
		return nil
	}
	cloned := make([]config.ReminderConfig, 0, len(reminders))
	cloned = append(cloned, reminders...)
	return cloned
}
