package engine

import (
	"errors"
	"time"

	"pause/internal/backend/domain/reminder"
	"pause/internal/backend/runtime/session"
	"pause/internal/backend/runtime/state"
	"pause/internal/logx"
)

func (e *Engine) canPostponeBreakLocked(view *state.BreakSessionView) bool {
	if view == nil || view.Status != string(session.StatusResting) || len(view.Reasons) == 0 {
		return false
	}
	if int(view.EndsAt.Sub(view.StartedAt).Seconds()) <= postponeBreakDelaySec {
		return false
	}
	for _, reason := range view.Reasons {
		if e.postponedOnce[reason] {
			return false
		}
	}
	return true
}

func (e *Engine) SkipCurrentBreak(now time.Time, mode SkipMode) (state.RuntimeState, error) {
	e.mu.Lock()
	var historyWrite *historyWrite

	settings := e.store.Get()
	view := e.session.CurrentView(now)
	switch mode {
	case "", SkipModeNormal:
		if !settings.Enforcement.OverlaySkipAllowed {
			e.mu.Unlock()
			return state.RuntimeState{}, errors.New("skip is disabled by settings")
		}
	case SkipModeEmergency:
		// Emergency path: allow explicit user escape from enforced overlay.
		e.session.SetCanSkip(true)
	default:
		e.mu.Unlock()
		return state.RuntimeState{}, errors.New("invalid skip mode")
	}

	if err := e.session.Skip(); err != nil {
		logx.Warnf("break.skip_err mode=%s err=%v", mode, err)
		e.mu.Unlock()
		return state.RuntimeState{}, err
	}
	if view != nil {
		historyWrite = e.prepareBreakSkippedWriteLocked(now, view)
		e.clearPostponedOnceForReasonsLocked(view.Reasons)
		logx.Infof(
			"break.skipped mode=%s reasons=%s remaining_sec=%d",
			mode,
			joinReasons(view.Reasons),
			view.RemainingSec,
		)
	}
	e.session.ClearIfDone()
	runtimeState := e.runtimeStateLocked(now, settings)
	e.mu.Unlock()
	e.commitHistoryWrite(historyWrite)
	return runtimeState, nil
}

func (e *Engine) PostponeCurrentBreak(now time.Time) (state.RuntimeState, error) {
	e.mu.Lock()
	settings := e.store.Get()
	view := e.session.CurrentView(now)
	if view == nil || view.Status != string(session.StatusResting) {
		e.mu.Unlock()
		return state.RuntimeState{}, errors.New("no active break")
	}
	if !e.canPostponeBreakLocked(view) {
		e.mu.Unlock()
		return state.RuntimeState{}, errors.New("break postpone is unavailable")
	}

	effectiveReminders := e.effectiveReminderConfigsLocked(e.reminders)
	nextByID := map[int64]reminder.Reminder{}
	for _, rem := range effectiveReminders {
		nextByID[rem.ID] = rem
	}
	postponed := make([]int64, 0, len(view.Reasons))
	for _, reason := range view.Reasons {
		rem, ok := nextByID[reason]
		if !ok || !rem.Enabled {
			continue
		}
		e.scheduler.PostponeByID(reason, rem.IntervalSec, postponeBreakDelaySec)
		e.postponedOnce[reason] = true
		postponed = append(postponed, reason)
	}
	if len(postponed) == 0 {
		e.mu.Unlock()
		return state.RuntimeState{}, errors.New("no active reminder rules")
	}

	e.session.Cancel()
	e.activeHistoryBreak = nil
	e.lastTick = now
	e.tickRemainder = 0
	runtimeState := e.runtimeStateLocked(now, settings)
	logx.Infof(
		"break.postponed reasons=%s delay_sec=%d",
		joinReasons(postponed),
		postponeBreakDelaySec,
	)
	e.mu.Unlock()
	return runtimeState, nil
}

func (e *Engine) StartBreakNow(now time.Time) (state.RuntimeState, error) {
	return e.StartBreakNowForReason(0, now)
}

func (e *Engine) StartBreakNowForReason(reason int64, now time.Time) (state.RuntimeState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	settings := e.store.Get()
	if !e.globalEnabled {
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
		reason,
	)
	return e.runtimeStateLocked(now, settings), nil
}
