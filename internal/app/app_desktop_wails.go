//go:build wails

package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"pause/internal/core/config"
	"pause/internal/core/service"
	"pause/internal/desktop"
	"pause/internal/logx"
)

type wailsDesktopController struct {
	lastOverlayActive       bool
	lastOverlaySkip         bool
	lastOverlayLang         string
	lastOverlayText         string
	lastOverlayTheme        string
	lastOverlaySessionStart time.Time
	overlayFallbackNotified bool
	lastRuntimeTick         time.Time
	smoothedNextEye         int
	smoothedNextStand       int
	hasSmoothedNext         bool
	lastSmoothKey           string
	lastLanguage            string
	statusBar               desktop.StatusBarController
	overlay                 desktop.BreakOverlayController
	startOnce               sync.Once
}

type autoReminderChoice struct {
	reason    string
	remaining int
	total     int
}

func newDesktopController() desktopController {
	return &wailsDesktopController{
		statusBar: desktop.NewStatusBarController(),
		overlay:   desktop.NewBreakOverlayController(),
	}
}

func (c *wailsDesktopController) OnStartup(ctx context.Context, app *App) {
	c.startOnce.Do(func() {
		logx.SetSink(func(level logx.Level, message string) {
			switch level {
			case logx.LevelError:
				runtime.LogError(ctx, message)
			case logx.LevelWarn:
				runtime.LogWarning(ctx, message)
			case logx.LevelInfo:
				runtime.LogInfo(ctx, message)
			default:
				runtime.LogDebug(ctx, message)
			}
		})
		go func() {
			<-ctx.Done()
			logx.ClearSink()
		}()

		desktop.ConfigureDesktopWindowBehavior()
		c.statusBar.Init(func(actionID int) {
			c.handleStatusBarAction(ctx, app, actionID)
		})
		c.overlay.Init(func() {
			_, err := app.skipCurrentBreakEmergency()
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
			c.syncOverlay(ctx, app, state, settings)
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
	c.statusBar.Update(status, countdown, title, state.Paused, progress)
}

func (c *wailsDesktopController) handleStatusBarAction(ctx context.Context, app *App, actionID int) {
	switch actionID {
	case desktop.StatusBarActionBreakNow:
		state := app.engine.GetRuntimeState(time.Now())
		choice := selectAutoReminderChoice(state, app.engine.GetSettings())
		_, err := app.StartBreakNowForReason(choice.reason)
		c.logErr(ctx, err)
	case desktop.StatusBarActionPause:
		_, err := app.Pause(service.PauseModeIndefinite, 0)
		c.logErr(ctx, err)
	case desktop.StatusBarActionPause30:
		_, err := app.Pause(service.PauseModeTemporary, 30*60)
		c.logErr(ctx, err)
	case desktop.StatusBarActionResume:
		_ = app.Resume()
	case desktop.StatusBarActionOpenWindow:
		desktop.ShowMainWindowFromStatusBar(ctx)
	case desktop.StatusBarActionQuit:
		app.Quit()
	}
}

func (c *wailsDesktopController) syncOverlay(ctx context.Context, app *App, state config.RuntimeState, settings config.Settings) {
	overlayActive := state.CurrentSession != nil && state.CurrentSession.Status == "resting"
	overlaySkipAllowed := overlayActive && state.OverlaySkipAllowed && state.CurrentSession != nil && state.CurrentSession.CanSkip
	language := c.lastLanguage
	theme := resolveEffectiveTheme(settings.UI.Theme)
	overlayText := ""
	if overlayActive && state.CurrentSession != nil {
		overlayText = overlayCountdownText(language, state.CurrentSession.RemainingSec)
		if !state.CurrentSession.StartedAt.Equal(c.lastOverlaySessionStart) {
			c.lastOverlaySessionStart = state.CurrentSession.StartedAt
			c.overlayFallbackNotified = false
		}
	} else {
		c.lastOverlaySessionStart = time.Time{}
		c.overlayFallbackNotified = false
	}

	if c.overlay.IsNative() {
		needsUpdate := overlayActive != c.lastOverlayActive || overlaySkipAllowed != c.lastOverlaySkip || language != c.lastOverlayLang || overlayText != c.lastOverlayText || theme != c.lastOverlayTheme
		if needsUpdate {
			if overlayActive {
				// Keep native break overlay isolated from the main window.
				desktop.HideMainWindowForOverlay(ctx)
				if !c.overlay.Show(overlaySkipAllowed, overlaySkipButtonTitle(language), overlayText, theme) {
					if !c.overlayFallbackNotified && app != nil {
						app.SendBreakFallbackNotification(state)
						c.overlayFallbackNotified = true
					}
				}
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
		if overlayActive {
			desktop.ShowMainWindowForOverlay(ctx)
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

func (c *wailsDesktopController) logErr(_ context.Context, err error) {
	if err == nil {
		return
	}
	logx.Errorf("desktop.error err=%v", err)
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
	choice := selectAutoReminderChoice(state, settings)
	if choice.reason == "" {
		if language == config.UILanguageZhCN {
			return "下次休息：关闭"
		}
		return "Next break: off"
	}

	reasonText := localizeReminderReason(choice.reason, language)

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
		choice := selectAutoReminderChoice(state, settings)
		if choice.reason == "" {
			return ""
		}
		return formatCountdown(choice.remaining)
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

	choice := selectAutoReminderChoice(state, settings)
	if choice.reason == "" {
		return ""
	}
	return formatCountdown(choice.remaining)
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
	choice := selectAutoReminderChoice(state, settings)
	if choice.reason == "" {
		return 0, 0
	}
	return choice.remaining, choice.total
}

func selectAutoReminderChoice(state config.RuntimeState, settings config.Settings) autoReminderChoice {
	nextEye := state.NextEyeInSec
	nextStand := state.NextStandInSec
	switch {
	case nextEye < 0 && nextStand < 0:
		return autoReminderChoice{}
	case nextEye >= 0 && nextStand >= 0:
		if nextEye <= nextStand {
			return autoReminderChoice{reason: "eye", remaining: nextEye, total: settings.Eye.IntervalSec}
		}
		return autoReminderChoice{reason: "stand", remaining: nextStand, total: settings.Stand.IntervalSec}
	case nextEye >= 0:
		return autoReminderChoice{reason: "eye", remaining: nextEye, total: settings.Eye.IntervalSec}
	default:
		return autoReminderChoice{reason: "stand", remaining: nextStand, total: settings.Stand.IntervalSec}
	}
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
