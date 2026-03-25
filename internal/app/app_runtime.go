package app

import (
	"errors"
	"time"

	"pause/internal/core/state"
)

func (a *App) GetRuntimeState() state.RuntimeState {
	runtimeState := a.engine.GetRuntimeState(time.Now())
	return a.decorateRuntimeState(runtimeState)
}

func (a *App) Pause() (state.RuntimeState, error) {
	runtimeState, err := a.engine.Pause(time.Now())
	if err != nil {
		return state.RuntimeState{}, err
	}
	return a.decorateRuntimeState(runtimeState), nil
}

func (a *App) Resume() state.RuntimeState {
	return a.decorateRuntimeState(a.engine.Resume(time.Now()))
}

func (a *App) PauseReminder(reminderID int64) (state.RuntimeState, error) {
	if reminderID <= 0 {
		return state.RuntimeState{}, errors.New("reminder id is required")
	}
	runtimeState, err := a.engine.PauseReminder(reminderID, time.Now())
	if err != nil {
		return state.RuntimeState{}, err
	}
	return a.decorateRuntimeState(runtimeState), nil
}

func (a *App) ResumeReminder(reminderID int64) (state.RuntimeState, error) {
	if reminderID <= 0 {
		return state.RuntimeState{}, errors.New("reminder id is required")
	}
	runtimeState, err := a.engine.ResumeReminder(reminderID, time.Now())
	if err != nil {
		return state.RuntimeState{}, err
	}
	return a.decorateRuntimeState(runtimeState), nil
}

func (a *App) SkipCurrentBreak() (state.RuntimeState, error) {
	return a.skipCurrentBreakWithMode(skipModeNormal)
}

func (a *App) skipCurrentBreakEmergency() (state.RuntimeState, error) {
	return a.skipCurrentBreakWithMode(skipModeEmergency)
}

func (a *App) skipCurrentBreakWithMode(mode skipMode) (state.RuntimeState, error) {
	runtimeState, err := a.engine.SkipCurrentBreak(time.Now(), mode)
	if err != nil {
		return state.RuntimeState{}, err
	}
	return a.decorateRuntimeState(runtimeState), nil
}

func (a *App) StartBreakNow() (state.RuntimeState, error) {
	runtimeState, err := a.engine.StartBreakNow(time.Now())
	if err != nil {
		return state.RuntimeState{}, err
	}
	return a.decorateRuntimeState(runtimeState), nil
}

func (a *App) StartBreakNowForReason(reminderID int64) (state.RuntimeState, error) {
	if reminderID < 0 {
		return state.RuntimeState{}, errors.New("reminder id is invalid")
	}
	runtimeState, err := a.engine.StartBreakNowForReason(reminderID, time.Now())
	if err != nil {
		return state.RuntimeState{}, err
	}
	return a.decorateRuntimeState(runtimeState), nil
}

func (a *App) decorateRuntimeState(state state.RuntimeState) state.RuntimeState {
	settings := a.engine.GetSettings()
	state.EffectiveLanguage = resolveEffectiveLanguage(settings.UI.Language)
	state.EffectiveTheme = resolveEffectiveTheme(settings.UI.Theme)
	return decorateRuntimeStateForPlatform(state)
}
