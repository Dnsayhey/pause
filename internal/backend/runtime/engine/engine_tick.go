package engine

import (
	"context"
	"time"

	"pause/internal/backend/domain/reminder"
	"pause/internal/backend/domain/settings"
	"pause/internal/backend/ports"
	"pause/internal/backend/runtime/scheduler"
	"pause/internal/backend/runtime/session"
	"pause/internal/logx"
)

const awayRestThreshold = time.Minute

type tickState struct {
	settings        settings.Settings
	reminders       []reminder.Reminder
	rawDeltaSec     int
	appliedDeltaSec int
}

func (e *Engine) Start(ctx context.Context) {
	e.startOnce.Do(func() {
		logx.Infof("engine.start")
		baseCtx := ctx
		if baseCtx == nil {
			baseCtx = context.Background()
		}
		runCtx, cancel := context.WithCancel(baseCtx)
		e.mu.Lock()
		e.runCtx = runCtx
		e.cancelRun = cancel
		e.currentLocked = e.lockProvider.IsScreenLocked()
		if e.currentLocked {
			e.screenLockedAt = e.lastTick
			if e.screenLockedAt.IsZero() {
				e.screenLockedAt = time.Now()
			}
		}
		e.closeLockEvents = e.lockProvider.SubscribeLockEvents(e.enqueueLockEvent)
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
				case <-runCtx.Done():
					return
				case now := <-ticker.C:
					e.Tick(now)
				}
			}
		}()
	})
}

func (e *Engine) Stop() {
	e.stopOnce.Do(func() {
		e.mu.Lock()
		cancel := e.cancelRun
		closeLockEvents := e.closeLockEvents
		e.cancelRun = nil
		e.runCtx = nil
		e.closeLockEvents = nil
		e.mu.Unlock()

		if closeLockEvents != nil {
			closeLockEvents()
		}
		if cancel != nil {
			cancel()
		}
		e.backgroundTasks.Wait()
	})
}

func (e *Engine) Tick(now time.Time) {
	idleSec := e.idleProvider.CurrentIdleSeconds()
	var historyWrite *historyWrite

	e.mu.Lock()

	firstTick := e.lastTick.IsZero()
	if firstTick {
		e.lastTick = now
	}

	tick := tickState{
		settings:  e.store.Get(),
		reminders: e.effectiveReminderConfigsLocked(e.reminders),
	}
	e.currentIdleSec = idleSec
	e.drainLockEventsLocked(tick.settings, tick.reminders)
	e.lastTickActive = e.isTickActive(tick.settings)
	tick.rawDeltaSec = int(now.Sub(e.lastTick).Seconds())
	tick.appliedDeltaSec = tick.rawDeltaSec
	if firstTick {
		e.logTickLocked(now, tick.settings, tick.reminders, "bootstrap", 0, 0, nil)
		e.mu.Unlock()
		return
	}

	historyWrite = e.completeFinishedSessionLocked(now, tick.settings)
	if e.stopTickLocked(now, tick, "global_disabled", !e.globalEnabled) {
		e.mu.Unlock()
		e.commitHistoryWrite(historyWrite)
		return
	}
	if e.stopTickLocked(now, tick, "session_active", e.session.IsActive()) {
		e.mu.Unlock()
		e.commitHistoryWrite(historyWrite)
		return
	}
	if e.stopTickLocked(now, tick, "idle_paused", !e.lastTickActive) {
		e.mu.Unlock()
		e.commitHistoryWrite(historyWrite)
		return
	}

	evt, reason, ok := e.advanceSchedulerLocked(now, &tick)
	if !ok {
		e.logTickLocked(now, tick.settings, tick.reminders, reason, tick.rawDeltaSec, tick.appliedDeltaSec, nil)
		e.mu.Unlock()
		e.commitHistoryWrite(historyWrite)
		return
	}
	e.dispatchScheduledEventLocked(now, tick, evt)
	e.mu.Unlock()
	e.commitHistoryWrite(historyWrite)
}

func (e *Engine) enqueueLockEvent(evt ports.LockEvent) {
	if evt.At.IsZero() {
		evt.At = time.Now()
	}
	select {
	case e.lockEvents <- evt:
	default:
		logx.Warnf("engine.lock_event_dropped kind=%s", evt.Kind)
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

func (e *Engine) drainLockEventsLocked(cfg settings.Settings, reminders []reminder.Reminder) {
	for {
		select {
		case evt := <-e.lockEvents:
			e.applyLockEventLocked(evt, cfg, reminders)
		default:
			return
		}
	}
}

func (e *Engine) applyLockEventLocked(evt ports.LockEvent, cfg settings.Settings, reminders []reminder.Reminder) {
	if evt.At.IsZero() {
		evt.At = time.Now()
	}
	switch evt.Kind {
	case ports.LockEventLocked:
		if e.currentLocked {
			return
		}
		e.currentLocked = true
		e.screenLockedAt = evt.At
		logx.Infof(
			"engine.screen_locked timer_mode=%s idle_sec=%d threshold_sec=%d",
			cfg.Timer.Mode,
			e.currentIdleSec,
			cfg.Timer.IdlePauseThresholdSec,
		)
	case ports.LockEventUnlocked:
		if !e.currentLocked {
			return
		}
		awayDuration := evt.At.Sub(e.screenLockedAt)
		if awayDuration < 0 {
			logx.Warnf("engine.lock_event_out_of_order kind=%s", evt.Kind)
			awayDuration = 0
		}
		e.currentLocked = false
		e.screenLockedAt = time.Time{}
		logx.Infof(
			"engine.screen_unlocked timer_mode=%s idle_sec=%d threshold_sec=%d",
			cfg.Timer.Mode,
			e.currentIdleSec,
			cfg.Timer.IdlePauseThresholdSec,
		)
		if cfg.Timer.Mode != settings.TimerModeRealTime || awayDuration >= awayRestThreshold {
			if evt.At.After(e.lastTick) {
				e.lastTick = evt.At
				e.tickRemainder = 0
			}
		}
		if awayDuration >= awayRestThreshold {
			resetRestReminderProgress(e.scheduler, reminders)
			logx.Infof(
				"engine.screen_away_rest_applied away_sec=%d threshold_sec=%d",
				int(awayDuration/time.Second),
				int(awayRestThreshold/time.Second),
			)
		}
	case "":
		return
	default:
		logx.Warnf("engine.lock_event_unknown kind=%s", evt.Kind)
	}
}

func (e *Engine) completeFinishedSessionLocked(now time.Time, cfg settings.Settings) *historyWrite {
	var historyWrite *historyWrite
	e.session.Tick(now)
	if view := e.session.CurrentView(now); view != nil && view.Status == string(session.StatusCompleted) {
		historyWrite = e.prepareBreakCompletedWriteLocked(view)
		e.clearPostponedOnceForReasonsLocked(view.Reasons)
		logx.Infof(
			"break.completed reasons=%s duration_sec=%d",
			joinReasons(view.Reasons),
			int(view.EndsAt.Sub(view.StartedAt).Seconds()),
		)
		if err := e.soundPlayer.PlayBreakEnd(cfg.Sound); err != nil {
			logx.Warnf("break.end_sound_err err=%v", err)
		}
	}
	e.session.ClearIfDone()
	return historyWrite
}

func (e *Engine) stopTickLocked(now time.Time, tick tickState, reason string, shouldStop bool) bool {
	if !shouldStop {
		return false
	}
	e.lastTick = now
	e.tickRemainder = 0
	e.logTickLocked(now, tick.settings, tick.reminders, reason, tick.rawDeltaSec, tick.appliedDeltaSec, nil)
	return true
}

func (e *Engine) advanceSchedulerLocked(now time.Time, tick *tickState) (*scheduler.Event, string, bool) {
	elapsed := now.Sub(e.lastTick) + e.tickRemainder
	if elapsed < 0 {
		e.lastTick = now
		e.tickRemainder = 0
		tick.appliedDeltaSec = 0
		return nil, "negative_elapsed", false
	}

	tick.rawDeltaSec = int(elapsed / time.Second)
	if tick.rawDeltaSec <= 0 {
		e.lastTick = now
		e.tickRemainder = elapsed
		tick.appliedDeltaSec = 0
		return nil, "sub_second_elapsed", false
	}

	tick.appliedDeltaSec = tick.rawDeltaSec
	e.tickRemainder = elapsed - (time.Duration(tick.appliedDeltaSec) * time.Second)

	evt := e.scheduler.OnActiveSeconds(tick.appliedDeltaSec, tick.reminders)
	e.lastTick = now
	if evt == nil {
		return nil, "no_event", false
	}
	return evt, "", true
}

func (e *Engine) dispatchScheduledEventLocked(now time.Time, tick tickState, evt *scheduler.Event) {
	restEvent, notifyReminderIDs := splitReminderEventByType(evt, tick.reminders)
	if len(notifyReminderIDs) > 0 {
		e.notifyRemindersLocked(notifyReminderIDs, resolveEffectiveLanguage(tick.settings.UI.Language))
	}
	if restEvent == nil {
		e.logTickLocked(now, tick.settings, tick.reminders, "notification_event", tick.rawDeltaSec, tick.appliedDeltaSec, evt)
		return
	}

	e.session.StartBreak(now, restEvent, tick.settings.Enforcement.OverlaySkipAllowed)
	e.recordBreakStartedLocked(now, "scheduled", restEvent)
	logx.Infof(
		"break.started source=scheduled reasons=%s break_sec=%d skip_allowed=%t",
		joinReminderTypes(restEvent.Reasons),
		restEvent.BreakSec,
		tick.settings.Enforcement.OverlaySkipAllowed,
	)
	e.logTickLocked(now, tick.settings, tick.reminders, "event", tick.rawDeltaSec, tick.appliedDeltaSec, restEvent)
}
