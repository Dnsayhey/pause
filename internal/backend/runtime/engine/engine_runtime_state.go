package engine

import (
	"time"

	"pause/internal/backend/domain/settings"
	"pause/internal/backend/runtime/state"
)

func (e *Engine) GetRuntimeState(now time.Time) state.RuntimeState {
	e.mu.Lock()
	defer e.mu.Unlock()

	settings := e.store.Get()
	return e.runtimeStateLocked(now, settings)
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
