package engine

import (
	"context"
	"errors"
	"time"

	"pause/internal/backend/domain/reminder"
	"pause/internal/backend/domain/settings"
	"pause/internal/backend/runtime/state"
	"pause/internal/logx"
)

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

func (e *Engine) SetReminderConfigs(reminders []reminder.Reminder) []reminder.Reminder {
	e.mu.Lock()
	defer e.mu.Unlock()

	next := cloneReminderConfigs(reminders)
	prev := cloneReminderConfigs(e.reminders)
	e.reminders = cloneReminderConfigs(next)
	e.applyReminderConfigPatchLocked(prev, next)
	logx.Infof("reminders.synced count=%d", len(e.reminders))
	return cloneReminderConfigs(e.reminders)
}

func (e *Engine) ApplyReminderSnapshot(ctx context.Context, reminders []reminder.Reminder) error {
	_ = ctx
	e.SetReminderConfigs(reminders)
	return nil
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

func (e *Engine) applyGlobalSettingPatchLocked(prev, next settings.Settings) {
	if prev.GlobalEnabled != next.GlobalEnabled {
		e.scheduler.Reset()
		e.pausedReminder = map[int64]bool{}
		e.lastTick = time.Time{}
		e.tickRemainder = 0
	}
}

func (e *Engine) applyReminderConfigPatchLocked(prev, next []reminder.Reminder) {
	prevByID := map[int64]reminder.Reminder{}
	for _, reminder := range prev {
		prevByID[reminder.ID] = reminder
	}
	nextByID := map[int64]reminder.Reminder{}
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

func (e *Engine) effectiveReminderConfigsLocked(reminders []reminder.Reminder) []reminder.Reminder {
	updated := make([]reminder.Reminder, 0, len(reminders))
	for _, reminder := range reminders {
		next := reminder
		if next.Enabled && e.pausedReminder[next.ID] {
			next.Enabled = false
		}
		updated = append(updated, next)
	}
	return updated
}
