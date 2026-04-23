package app

import (
	"errors"
	"time"

	runtimestate "pause/internal/backend/runtime/state"
)

func (a *App) GetRuntimeState() RuntimeState {
	runtimeState := a.engine.GetRuntimeState(time.Now())
	return a.decorateRuntimeState(runtimeState)
}

func (a *App) Pause() (RuntimeState, error) {
	return a.decorateRuntimeState(a.engine.Pause(time.Now())), nil
}

func (a *App) Resume() RuntimeState {
	return a.decorateRuntimeState(a.engine.Resume(time.Now()))
}

func (a *App) PauseReminder(reminderID int64) (RuntimeState, error) {
	if reminderID <= 0 {
		return RuntimeState{}, errors.New("reminder id is required")
	}
	runtimeState, err := a.engine.PauseReminder(reminderID, time.Now())
	if err != nil {
		return RuntimeState{}, err
	}
	return a.decorateRuntimeState(runtimeState), nil
}

func (a *App) ResumeReminder(reminderID int64) (RuntimeState, error) {
	if reminderID <= 0 {
		return RuntimeState{}, errors.New("reminder id is required")
	}
	runtimeState, err := a.engine.ResumeReminder(reminderID, time.Now())
	if err != nil {
		return RuntimeState{}, err
	}
	return a.decorateRuntimeState(runtimeState), nil
}

func (a *App) SkipCurrentBreak() (RuntimeState, error) {
	return a.skipCurrentBreakWithMode(skipModeNormal)
}

func (a *App) skipCurrentBreakEmergency() (RuntimeState, error) {
	return a.skipCurrentBreakWithMode(skipModeEmergency)
}

func (a *App) skipCurrentBreakWithMode(mode skipMode) (RuntimeState, error) {
	runtimeState, err := a.engine.SkipCurrentBreak(time.Now(), mode)
	if err != nil {
		return RuntimeState{}, err
	}
	return a.decorateRuntimeState(runtimeState), nil
}

func (a *App) StartBreakNow() (RuntimeState, error) {
	runtimeState, err := a.engine.StartBreakNow(time.Now())
	if err != nil {
		return RuntimeState{}, err
	}
	return a.decorateRuntimeState(runtimeState), nil
}

func (a *App) StartBreakNowForReason(reminderID int64) (RuntimeState, error) {
	if reminderID < 0 {
		return RuntimeState{}, errors.New("reminder id is invalid")
	}
	runtimeState, err := a.engine.StartBreakNowForReason(reminderID, time.Now())
	if err != nil {
		return RuntimeState{}, err
	}
	return a.decorateRuntimeState(runtimeState), nil
}

func (a *App) decorateRuntimeState(runtimeState runtimestate.RuntimeState) RuntimeState {
	settings := a.engine.GetSettings()
	runtimeState.EffectiveLanguage = resolveEffectiveLanguage(settings.UI.Language)
	runtimeState.EffectiveTheme = resolveEffectiveTheme(settings.UI.Theme)
	return decorateRuntimeStateForPlatform(runtimeStateFromDomain(runtimeState))
}
