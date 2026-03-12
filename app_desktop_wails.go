//go:build wails

package main

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"pause/internal/config"
	"pause/internal/diag"
	"pause/internal/service"
)

type wailsDesktopController struct {
	lastOverlayActive bool
	lastOverlaySkip   bool
	lastOverlayLang   string
	lastOverlayText   string
	lastOverlayTheme  string
	lastRuntimeTick   time.Time
	smoothedNextEye   int
	smoothedNextStand int
	hasSmoothedNext   bool
	lastSmoothKey     string
	lastLanguage      string
	statusBar         statusBarController
	overlay           breakOverlayController
	startOnce         sync.Once
}

func newDesktopController() desktopController {
	return &wailsDesktopController{
		statusBar: newStatusBarController(),
		overlay:   newBreakOverlayController(),
	}
}

func (c *wailsDesktopController) OnStartup(ctx context.Context, app *App) {
	c.startOnce.Do(func() {
		diag.Logf("desktop.start")
		configureDesktopWindowBehavior()
		c.statusBar.Init(func(actionID int) {
			c.handleStatusBarAction(ctx, app, actionID)
		})
		c.overlay.Init(func() {
			_, err := app.SkipCurrentBreak()
			c.logErr(ctx, err)
		})
		settings := app.engine.GetSettings()
		c.lastLanguage = resolveEffectiveLanguage(settings.UI.Language)
		c.statusBar.SetLocale(buildStatusBarLocaleStrings(c.lastLanguage))
		go c.runtimeLoop(ctx, app)
	})
}

func (c *wailsDesktopController) runtimeLoop(ctx context.Context, app *App) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	defer c.statusBar.Destroy()
	defer c.overlay.Destroy()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			gapMs := int64(0)
			if !c.lastRuntimeTick.IsZero() {
				gapMs = now.Sub(c.lastRuntimeTick).Milliseconds()
			}
			c.lastRuntimeTick = now

			settings := app.engine.GetSettings()
			state := app.engine.GetRuntimeState(now)
			c.syncStatusBar(state, settings, gapMs)
			c.syncOverlay(ctx, state, settings)
		}
	}
}

func (c *wailsDesktopController) syncStatusBar(state config.RuntimeState, settings config.Settings, loopGapMs int64) {
	language := resolveEffectiveLanguage(settings.UI.Language)
	if c.lastLanguage != language {
		c.statusBar.SetLocale(buildStatusBarLocaleStrings(language))
		c.lastLanguage = language
	}
	smoothKey := buildSmoothKey(settings)
	if c.lastSmoothKey != smoothKey {
		c.hasSmoothedNext = false
		c.lastSmoothKey = smoothKey
	}

	displayNextEye, displayNextStand := c.smoothCountdown(state.NextEyeInSec, state.NextStandInSec, loopGapMs)
	displayState := state
	displayState.NextEyeInSec = displayNextEye
	displayState.NextStandInSec = displayNextStand

	status := buildPauseLabel(displayState, language)
	countdown := buildCountdownLabel(displayState, settings, language)
	title := buildStatusBarTitle(displayState, settings, language)
	progress := buildStatusBarProgress(displayState, settings)
	started := time.Now()
	c.statusBar.Update(status, countdown, title, state.Paused, progress)
	updateMs := time.Since(started).Milliseconds()

	sessionStatus := "none"
	if state.CurrentSession != nil {
		sessionStatus = state.CurrentSession.Status
	}
	diag.Logf(
		"desktop.tick gap_ms=%d update_ms=%d next_eye_raw=%d next_stand_raw=%d next_eye_display=%d next_stand_display=%d paused=%t session=%s title=%q countdown=%q progress=%.3f",
		loopGapMs,
		updateMs,
		state.NextEyeInSec,
		state.NextStandInSec,
		displayNextEye,
		displayNextStand,
		state.Paused,
		sessionStatus,
		title,
		countdown,
		progress,
	)
}

func (c *wailsDesktopController) handleStatusBarAction(ctx context.Context, app *App, actionID int) {
	diag.Logf("desktop.action id=%d", actionID)
	switch actionID {
	case statusBarActionBreakNow:
		_, err := app.StartBreakNow()
		c.logErr(ctx, err)
	case statusBarActionPause:
		_, err := app.Pause(service.PauseModeIndefinite, 0)
		c.logErr(ctx, err)
	case statusBarActionPause30:
		_, err := app.Pause(service.PauseModeTemporary, 30*60)
		c.logErr(ctx, err)
	case statusBarActionResume:
		_ = app.Resume()
	case statusBarActionOpenWindow:
		showMainWindowFromStatusBar(ctx)
	case statusBarActionQuit:
		app.Quit()
	}
}

func (c *wailsDesktopController) syncOverlay(ctx context.Context, state config.RuntimeState, settings config.Settings) {
	overlayActive := state.CurrentSession != nil && state.CurrentSession.Status == "resting" && state.OverlayEnabled
	overlaySkipAllowed := overlayActive && state.OverlaySkipAllowed && state.CurrentSession != nil && state.CurrentSession.CanSkip
	language := c.lastLanguage
	theme := resolveEffectiveTheme(settings.UI.Theme)
	overlayText := ""
	if overlayActive && state.CurrentSession != nil {
		overlayText = overlayCountdownText(language, state.CurrentSession.RemainingSec)
	}

	if c.overlay.IsNative() {
		needsUpdate := overlayActive != c.lastOverlayActive || overlaySkipAllowed != c.lastOverlaySkip || language != c.lastOverlayLang || overlayText != c.lastOverlayText || theme != c.lastOverlayTheme
		if needsUpdate {
			diag.Logf("desktop.overlay native active=%t allow_skip=%t text=%q theme=%s", overlayActive, overlaySkipAllowed, overlayText, theme)
			if overlayActive {
				c.overlay.Show(overlaySkipAllowed, overlaySkipButtonTitle(language), overlayText, theme)
			} else {
				c.overlay.Hide()
			}
		}
		c.lastOverlayActive = overlayActive
		c.lastOverlaySkip = overlaySkipAllowed
		c.lastOverlayLang = language
		c.lastOverlayText = overlayText
		c.lastOverlayTheme = theme
		return
	}

	if overlayActive != c.lastOverlayActive {
		diag.Logf("desktop.overlay fallback active=%t", overlayActive)
		if overlayActive {
			showMainWindowForOverlay(ctx)
			runtime.WindowSetAlwaysOnTop(ctx, true)
			runtime.EventsEmit(ctx, "break:overlay", map[string]any{"active": true})
		} else {
			runtime.WindowSetAlwaysOnTop(ctx, false)
			runtime.EventsEmit(ctx, "break:overlay", map[string]any{"active": false})
		}
	}
	c.lastOverlayActive = overlayActive
	c.lastOverlaySkip = overlaySkipAllowed
	c.lastOverlayLang = language
	c.lastOverlayText = overlayText
	c.lastOverlayTheme = theme
}

func (c *wailsDesktopController) logErr(ctx context.Context, err error) {
	if err == nil || ctx == nil {
		return
	}
	diag.Logf("desktop.error err=%v", err)
	runtime.LogErrorf(ctx, "menu action failed: %v", err)
}

func buildPauseLabel(state config.RuntimeState, language string) string {
	if !state.GlobalEnabled {
		if language == config.UILanguageZhCN {
			return "状态：已关闭"
		}
		return "Status: disabled"
	}
	if state.Paused {
		if state.PauseMode == service.PauseModeIndefinite {
			if language == config.UILanguageZhCN {
				return "状态：已暂停（无限时）"
			}
			return "Status: paused (indefinite)"
		}
		if language == config.UILanguageZhCN {
			return "状态：已暂停"
		}
		return "Status: paused"
	}
	if state.CurrentSession != nil && state.CurrentSession.Status == "resting" {
		if language == config.UILanguageZhCN {
			return "状态：休息中"
		}
		return "Status: on break"
	}
	if language == config.UILanguageZhCN {
		return "状态：运行中"
	}
	return "Status: running"
}

func buildCountdownLabel(state config.RuntimeState, settings config.Settings, language string) string {
	_ = settings

	nextEye := state.NextEyeInSec
	nextStand := state.NextStandInSec
	if nextEye < 0 && nextStand < 0 {
		if language == config.UILanguageZhCN {
			return "下次休息：关闭"
		}
		return "Next break: off"
	}

	reasonText := localizeReminderReason("stand", language)
	if nextEye >= 0 && nextStand >= 0 {
		if int(math.Abs(float64(nextEye-nextStand))) <= 60 {
			if language == config.UILanguageZhCN {
				reasonText = fmt.Sprintf("%s+%s", localizeReminderReason("eye", language), localizeReminderReason("stand", language))
			} else {
				reasonText = "eye+stand"
			}
		} else if nextEye < nextStand {
			reasonText = localizeReminderReason("eye", language)
		} else {
			reasonText = localizeReminderReason("stand", language)
		}
	} else if nextEye >= 0 {
		reasonText = localizeReminderReason("eye", language)
	} else {
		reasonText = localizeReminderReason("stand", language)
	}

	if language == config.UILanguageZhCN {
		return fmt.Sprintf("下次休息：%s", reasonText)
	}
	return fmt.Sprintf("Next break: %s", reasonText)
}

func formatCountdown(sec int) string {
	if sec < 0 {
		return "off"
	}
	m := sec / 60
	s := sec % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *wailsDesktopController) smoothCountdown(nextEye, nextStand int, loopGapMs int64) (int, int) {
	if !c.hasSmoothedNext {
		c.smoothedNextEye = nextEye
		c.smoothedNextStand = nextStand
		c.hasSmoothedNext = true
		return nextEye, nextStand
	}

	stepSec := 1
	if loopGapMs > 1500 {
		stepSec = int((loopGapMs + 500) / 1000)
		if stepSec < 1 {
			stepSec = 1
		}
	}

	c.smoothedNextEye = smoothCountdownValue(c.smoothedNextEye, nextEye, stepSec)
	c.smoothedNextStand = smoothCountdownValue(c.smoothedNextStand, nextStand, stepSec)
	return c.smoothedNextEye, c.smoothedNextStand
}

func smoothCountdownValue(previous, target, stepSec int) int {
	if target < 0 {
		return target
	}
	if previous < 0 {
		return target
	}
	if stepSec < 1 {
		stepSec = 1
	}

	// Expected visual progression at normal speed.
	expected := previous - stepSec
	if expected < 0 {
		expected = 0
	}
	if target >= expected {
		// Snap upward on resets or pauses to keep visible state honest.
		return target
	}
	lag := expected - target
	snapThreshold := stepSec * 4
	if snapThreshold < 10 {
		snapThreshold = 10
	}
	if lag > snapThreshold {
		// Large downward jumps usually mean settings changed; avoid stale countdown drift.
		return target
	}

	// Backend moved faster than visible state; catch up smoothly by one extra second.
	catchup := expected - 1
	if catchup < 0 {
		catchup = 0
	}
	if catchup < target {
		return target
	}
	return catchup
}

func buildSmoothKey(settings config.Settings) string {
	return fmt.Sprintf(
		"%t|%t|%d|%t|%d|%s|%d",
		settings.GlobalEnabled,
		settings.Eye.Enabled,
		settings.Eye.IntervalSec,
		settings.Stand.Enabled,
		settings.Stand.IntervalSec,
		settings.Timer.Mode,
		settings.Timer.IdlePauseThresholdSec,
	)
}

func buildStatusBarTitle(state config.RuntimeState, settings config.Settings, _ string) string {
	if !settings.UI.ShowTrayCountdown {
		return ""
	}

	if state.Paused {
		nextEye := state.NextEyeInSec
		nextStand := state.NextStandInSec
		if nextEye < 0 && nextStand < 0 {
			return ""
		}
		if nextEye >= 0 && nextStand >= 0 {
			return formatCountdown(minInt(nextEye, nextStand))
		}
		if nextEye >= 0 {
			return formatCountdown(nextEye)
		}
		return formatCountdown(nextStand)
	}

	if state.CurrentSession != nil && state.CurrentSession.Status == "resting" {
		if state.CurrentSession.RemainingSec < 0 {
			return "00:00"
		}
		return formatCountdown(state.CurrentSession.RemainingSec)
	}

	if !state.GlobalEnabled {
		return ""
	}

	nextEye := state.NextEyeInSec
	nextStand := state.NextStandInSec
	if nextEye < 0 && nextStand < 0 {
		return ""
	}
	if nextEye >= 0 && nextStand >= 0 {
		return formatCountdown(minInt(nextEye, nextStand))
	}
	if nextEye >= 0 {
		return formatCountdown(nextEye)
	}
	return formatCountdown(nextStand)
}

func buildStatusBarProgress(state config.RuntimeState, settings config.Settings) float64 {
	if state.CurrentSession != nil && state.CurrentSession.Status == "resting" {
		// During break keep progress bar fixed; next countdown starts after break ends.
		return 1
	}

	if !state.GlobalEnabled {
		return 0
	}

	remaining, total := nearestCountdownRemainingAndTotal(state, settings)
	if total <= 0 {
		return 0
	}
	progress := 1 - (float64(remaining) / float64(total))
	return clampProgress(progress)
}

func nearestCountdownRemainingAndTotal(state config.RuntimeState, settings config.Settings) (int, int) {
	nextEye := state.NextEyeInSec
	nextStand := state.NextStandInSec
	if nextEye < 0 && nextStand < 0 {
		return 0, 0
	}

	if nextEye >= 0 && nextStand >= 0 {
		if int(math.Abs(float64(nextEye-nextStand))) <= 60 {
			return minInt(nextEye, nextStand), minInt(settings.Eye.IntervalSec, settings.Stand.IntervalSec)
		}
		if nextEye < nextStand {
			return nextEye, settings.Eye.IntervalSec
		}
		return nextStand, settings.Stand.IntervalSec
	}

	if nextEye >= 0 {
		return nextEye, settings.Eye.IntervalSec
	}
	return nextStand, settings.Stand.IntervalSec
}

func clampProgress(progress float64) float64 {
	if progress < 0 {
		return 0
	}
	if progress > 1 {
		return 1
	}
	return progress
}
