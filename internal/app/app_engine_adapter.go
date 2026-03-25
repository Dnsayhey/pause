package app

import (
	"context"
	"time"

	"pause/internal/core/reminder"
	"pause/internal/core/service"
	"pause/internal/core/settings"
	"pause/internal/core/state"
)

type coreEngineAdapter struct {
	engine *service.Engine
}

func newEngineRuntime(engine *service.Engine) engineRuntime {
	if engine == nil {
		return nil
	}
	return &coreEngineAdapter{engine: engine}
}

func (a *coreEngineAdapter) Start(ctx context.Context) {
	a.engine.Start(ctx)
}

func (a *coreEngineAdapter) GetSettings() settings.Settings {
	return a.engine.GetSettings()
}

func (a *coreEngineAdapter) GetRuntimeState(now time.Time) state.RuntimeState {
	return a.engine.GetRuntimeState(now)
}

func (a *coreEngineAdapter) Pause(now time.Time) (state.RuntimeState, error) {
	return a.engine.Pause(now)
}

func (a *coreEngineAdapter) Resume(now time.Time) state.RuntimeState {
	return a.engine.Resume(now)
}

func (a *coreEngineAdapter) PauseReminder(reminderID int64, now time.Time) (state.RuntimeState, error) {
	return a.engine.PauseReminder(reminderID, now)
}

func (a *coreEngineAdapter) ResumeReminder(reminderID int64, now time.Time) (state.RuntimeState, error) {
	return a.engine.ResumeReminder(reminderID, now)
}

func (a *coreEngineAdapter) SkipCurrentBreak(now time.Time, mode skipMode) (state.RuntimeState, error) {
	return a.engine.SkipCurrentBreak(now, service.SkipMode(mode))
}

func (a *coreEngineAdapter) StartBreakNow(now time.Time) (state.RuntimeState, error) {
	return a.engine.StartBreakNow(now)
}

func (a *coreEngineAdapter) StartBreakNowForReason(reason int64, now time.Time) (state.RuntimeState, error) {
	return a.engine.StartBreakNowForReason(reason, now)
}

func (a *coreEngineAdapter) SetReminderConfigs(reminders []reminder.ReminderConfig) []reminder.ReminderConfig {
	return a.engine.SetReminderConfigs(reminders)
}
