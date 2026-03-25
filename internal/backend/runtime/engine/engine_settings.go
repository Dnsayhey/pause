package engine

import (
	"time"

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

func (e *Engine) applyGlobalSettingPatchLocked(prev, next settings.Settings) {
	if prev.GlobalEnabled != next.GlobalEnabled {
		e.scheduler.Reset()
		e.pausedReminder = map[int64]bool{}
		e.lastTick = time.Time{}
		e.tickRemainder = 0
	}
}
