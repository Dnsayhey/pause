package engine

import (
	"context"
	"time"

	"pause/internal/backend/domain/reminder"
	"pause/internal/backend/domain/settings"
	"pause/internal/backend/runtime/scheduler"
	"pause/internal/backend/runtime/session"
	"pause/internal/logx"
)

type tickState struct {
	settings        settings.Settings
	reminders       []reminder.Reminder
	rawDeltaSec     int
	appliedDeltaSec int
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

	tick := tickState{
		settings:  e.store.Get(),
		reminders: e.effectiveReminderConfigsLocked(e.reminders),
	}
	e.updateTickActivityLocked(tick.settings, idleSec, locked)
	tick.rawDeltaSec = int(now.Sub(e.lastTick).Seconds())
	tick.appliedDeltaSec = tick.rawDeltaSec
	if firstTick {
		e.logTickLocked(now, tick.settings, tick.reminders, "bootstrap", 0, 0, nil)
		return
	}

	e.completeFinishedSessionLocked(now, tick.settings)
	if e.stopTickLocked(now, tick, "global_disabled", !e.globalEnabled) {
		return
	}
	if e.stopTickLocked(now, tick, "session_active", e.session.IsActive()) {
		return
	}
	if e.stopTickLocked(now, tick, "idle_paused", !e.lastTickActive) {
		return
	}

	evt, reason, ok := e.advanceSchedulerLocked(now, &tick)
	if !ok {
		e.logTickLocked(now, tick.settings, tick.reminders, reason, tick.rawDeltaSec, tick.appliedDeltaSec, nil)
		return
	}
	e.dispatchScheduledEventLocked(now, tick, evt)
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

func (e *Engine) updateTickActivityLocked(cfg settings.Settings, idleSec int, locked bool) {
	wasLocked := e.currentLocked
	e.currentIdleSec = idleSec
	e.currentLocked = locked
	if !wasLocked && locked {
		logx.Infof(
			"engine.screen_locked timer_mode=%s idle_sec=%d threshold_sec=%d",
			cfg.Timer.Mode,
			idleSec,
			cfg.Timer.IdlePauseThresholdSec,
		)
	} else if wasLocked && !locked {
		logx.Infof(
			"engine.screen_unlocked timer_mode=%s idle_sec=%d threshold_sec=%d",
			cfg.Timer.Mode,
			idleSec,
			cfg.Timer.IdlePauseThresholdSec,
		)
	}
	e.lastTickActive = e.isTickActive(cfg)
}

func (e *Engine) completeFinishedSessionLocked(now time.Time, cfg settings.Settings) {
	e.session.Tick(now)
	if view := e.session.CurrentView(now); view != nil && view.Status == string(session.StatusCompleted) {
		e.recordBreakCompletedLocked(view)
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
