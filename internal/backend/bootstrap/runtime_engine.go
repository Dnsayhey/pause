package bootstrap

import (
	"context"
	"time"

	"pause/internal/backend/domain/settings"
	service "pause/internal/backend/runtime/engine"
	"pause/internal/backend/runtime/state"
)

type SkipMode = service.SkipMode

const (
	SkipModeNormal    SkipMode = service.SkipModeNormal
	SkipModeEmergency SkipMode = service.SkipModeEmergency
)

type RuntimeEngine interface {
	Start(ctx context.Context)
	Stop()
	GetSettings() settings.Settings
	GetRuntimeState(now time.Time) state.RuntimeState
	Pause(now time.Time) state.RuntimeState
	Resume(now time.Time) state.RuntimeState
	PauseReminder(reminderID int64, now time.Time) (state.RuntimeState, error)
	ResumeReminder(reminderID int64, now time.Time) (state.RuntimeState, error)
	SkipCurrentBreak(now time.Time, mode SkipMode) (state.RuntimeState, error)
	PostponeCurrentBreak(now time.Time) (state.RuntimeState, error)
	StartBreakNow(now time.Time) (state.RuntimeState, error)
	StartBreakNowForReason(reason int64, now time.Time) (state.RuntimeState, error)
}

type runtimeEngineAdapter struct {
	engine *service.Engine
}

func WrapEngine(engine *service.Engine) RuntimeEngine {
	if engine == nil {
		return nil
	}
	return &runtimeEngineAdapter{engine: engine}
}

func (a *runtimeEngineAdapter) Start(ctx context.Context) {
	a.engine.Start(ctx)
}

func (a *runtimeEngineAdapter) Stop() {
	a.engine.Stop()
}

func (a *runtimeEngineAdapter) GetSettings() settings.Settings {
	return a.engine.GetSettings()
}

func (a *runtimeEngineAdapter) GetRuntimeState(now time.Time) state.RuntimeState {
	return a.engine.GetRuntimeState(now)
}

func (a *runtimeEngineAdapter) Pause(now time.Time) state.RuntimeState {
	return a.engine.Pause(now)
}

func (a *runtimeEngineAdapter) Resume(now time.Time) state.RuntimeState {
	return a.engine.Resume(now)
}

func (a *runtimeEngineAdapter) PauseReminder(reminderID int64, now time.Time) (state.RuntimeState, error) {
	return a.engine.PauseReminder(reminderID, now)
}

func (a *runtimeEngineAdapter) ResumeReminder(reminderID int64, now time.Time) (state.RuntimeState, error) {
	return a.engine.ResumeReminder(reminderID, now)
}

func (a *runtimeEngineAdapter) SkipCurrentBreak(now time.Time, mode SkipMode) (state.RuntimeState, error) {
	return a.engine.SkipCurrentBreak(now, mode)
}

func (a *runtimeEngineAdapter) PostponeCurrentBreak(now time.Time) (state.RuntimeState, error) {
	return a.engine.PostponeCurrentBreak(now)
}

func (a *runtimeEngineAdapter) StartBreakNow(now time.Time) (state.RuntimeState, error) {
	return a.engine.StartBreakNow(now)
}

func (a *runtimeEngineAdapter) StartBreakNowForReason(reason int64, now time.Time) (state.RuntimeState, error) {
	return a.engine.StartBreakNowForReason(reason, now)
}
