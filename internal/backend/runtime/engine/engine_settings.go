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
	patchJSON := marshalPatchForLog(patch)
	next, err := e.store.Update(patch)
	if err != nil {
		logx.Warnf("settings.update_err patch=%s err=%v", patchJSON, err)
		return settings.Settings{}, err
	}

	if patch.Enforcement != nil && patch.Enforcement.OverlaySkipAllowed != nil {
		e.mu.Lock()
		e.session.SetCanSkip(next.Enforcement.OverlaySkipAllowed)
		e.mu.Unlock()
	}

	logx.Infof("settings.updated patch=%s", patchJSON)

	return next, nil
}

func (e *Engine) Pause(now time.Time) state.RuntimeState {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.setGlobalEnabledLocked(false, now)
	logx.Infof("runtime_global_enabled.set enabled=false source=pause")
	return e.runtimeStateLocked(now, e.store.Get())
}

func (e *Engine) Resume(now time.Time) state.RuntimeState {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.setGlobalEnabledLocked(true, now)
	logx.Infof("runtime_global_enabled.set enabled=true source=resume")
	return e.runtimeStateLocked(now, e.store.Get())
}

func (e *Engine) setGlobalEnabledLocked(enabled bool, now time.Time) {
	if e.globalEnabled == enabled {
		return
	}
	e.globalEnabled = enabled
	// Freeze scheduler progress while disabled without discarding elapsed state.
	// Also reset tick baseline to prevent catch-up over paused duration.
	e.lastTick = now
	e.tickRemainder = 0
}
