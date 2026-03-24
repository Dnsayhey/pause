//go:build wails

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
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
	lastLanguage            string
	reminderActionOrder     []string
	reminderOrderMu         sync.RWMutex
	lastStatusBarStatus     string
	lastStatusBarCountdown  string
	lastStatusBarTitle      string
	lastStatusBarPaused     bool
	lastRemindersPayload    string
	lastDetailsVisible      bool
	hasStatusBarSnapshot    bool
	statusBarSyncMu         sync.Mutex
	statusBarDetailsVisible atomic.Bool
	statusBar               desktop.StatusBarController
	overlay                 desktop.BreakOverlayController
	startOnce               sync.Once
}

type autoReminderChoice struct {
	reason    string
	name      string
	remaining int
	total     int
}

type reminderStatusView struct {
	Reason   string  `json:"reason"`
	Paused   bool    `json:"paused"`
	Title    string  `json:"title"`
	Progress float64 `json:"progress"`
}

func newDesktopController() desktopController {
	controller := &wailsDesktopController{
		statusBar: desktop.NewStatusBarController(),
		overlay:   desktop.NewBreakOverlayController(),
	}
	controller.statusBarDetailsVisible.Store(true)
	return controller
}

func overlaySkipMode(settings config.Settings) service.SkipMode {
	if settings.Enforcement.OverlaySkipAllowed {
		return service.SkipModeNormal
	}
	return service.SkipModeEmergency
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
			shutdownPreferredThemeProvider()
			logx.ClearSink()
		}()

		initPreferredThemeProvider()
		desktop.ConfigureDesktopWindowBehavior()
		c.statusBar.Init(func(event desktop.StatusBarEvent) {
			c.handleStatusBarEvent(ctx, app, event)
		})
		c.overlay.Init(func() {
			skipMode := overlaySkipMode(app.engine.GetSettings())
			_, err := app.skipCurrentBreakWithMode(skipMode)
			c.logErr(ctx, err)
		})
		settings := app.engine.GetSettings()
		c.lastLanguage = resolveEffectiveLanguage(settings.UI.Language)
		c.statusBar.SetLocale(buildStatusBarLocaleStrings(c.lastLanguage))
		go c.runtimeLoop(ctx, app)
	})
}

func (c *wailsDesktopController) runtimeLoop(ctx context.Context, app *App) {
	const statusLoopOffset = 150 * time.Millisecond
	offsetTimer := time.NewTimer(statusLoopOffset)
	defer offsetTimer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-offsetTimer.C:
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	defer c.statusBar.Destroy()
	defer c.overlay.Destroy()

	settings := app.engine.GetSettings()
	state := app.engine.GetRuntimeState(time.Now())
	c.syncStatusBarWithLock(state, settings)
	c.syncOverlay(ctx, app, state, settings)

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			settings := app.engine.GetSettings()
			state := app.engine.GetRuntimeState(now)
			c.syncStatusBarWithLock(state, settings)
			c.syncOverlay(ctx, app, state, settings)
		}
	}
}

func (c *wailsDesktopController) syncStatusBarWithLock(state config.RuntimeState, settings config.Settings) {
	c.statusBarSyncMu.Lock()
	defer c.statusBarSyncMu.Unlock()
	c.syncStatusBar(state, settings)
}

func (c *wailsDesktopController) syncStatusBar(state config.RuntimeState, settings config.Settings) {
	language := resolveEffectiveLanguage(settings.UI.Language)
	if c.lastLanguage != language {
		c.statusBar.SetLocale(buildStatusBarLocaleStrings(language))
		c.lastLanguage = language
	}

	status := buildPauseLabel(state, language)
	countdown := buildCountdownLabel(state, language)
	title := buildStatusBarTitle(state)
	paused := !state.GlobalEnabled
	detailsVisible := c.statusBarDetailsVisible.Load()
	remindersPayload := ""
	if detailsVisible {
		payload, reminderOrder := buildRemindersPayload(state, language)
		remindersPayload = payload
		c.setReminderActionOrder(reminderOrder)
	}

	if c.hasStatusBarSnapshot &&
		c.lastStatusBarStatus == status &&
		c.lastStatusBarCountdown == countdown &&
		c.lastStatusBarTitle == title &&
		c.lastStatusBarPaused == paused &&
		c.lastRemindersPayload == remindersPayload &&
		c.lastDetailsVisible == detailsVisible {
		return
	}

	progress := buildStatusBarProgress(state)
	c.statusBar.Update(status, countdown, title, paused, progress, remindersPayload)
	c.lastStatusBarStatus = status
	c.lastStatusBarCountdown = countdown
	c.lastStatusBarTitle = title
	c.lastStatusBarPaused = paused
	c.lastRemindersPayload = remindersPayload
	c.lastDetailsVisible = detailsVisible
	c.hasStatusBarSnapshot = true
}

func (c *wailsDesktopController) handleStatusBarEvent(ctx context.Context, app *App, event desktop.StatusBarEvent) {
	switch event.Kind {
	case desktop.StatusBarEventAction:
		c.handleStatusBarAction(ctx, app, event.ActionID)
	case desktop.StatusBarEventVisibilityChanged:
		c.statusBarDetailsVisible.Store(event.Visible)
		if event.Visible {
			settings := app.engine.GetSettings()
			state := app.engine.GetRuntimeState(time.Now())
			c.syncStatusBarWithLock(state, settings)
		}
	}
}

func (c *wailsDesktopController) handleStatusBarAction(ctx context.Context, app *App, actionID int) {
	switch actionID {
	case desktop.StatusBarActionBreakNow:
		state := app.engine.GetRuntimeState(time.Now())
		choice := selectAutoReminderChoice(state)
		_, err := app.StartBreakNowForReason(choice.reason)
		c.logErr(ctx, err)
	case desktop.StatusBarActionPause:
		_, err := app.Pause()
		c.logErr(ctx, err)
	case desktop.StatusBarActionResume:
		_ = app.Resume()
	case desktop.StatusBarActionOpenWindow:
		desktop.ShowMainWindowFromStatusBar(ctx)
	case desktop.StatusBarActionQuit:
		app.Quit()
	default:
		if actionID >= desktop.StatusBarActionPauseReminderBase && actionID < desktop.StatusBarActionResumeReminderBase {
			row := actionID - desktop.StatusBarActionPauseReminderBase
			if reason, ok := c.reminderReasonByRow(row); ok {
				logx.Infof("desktop.statusbar_action action=pause_reminder row=%d reason=%s", row, reason)
				_, err := app.PauseReminder(reason)
				c.logErr(ctx, err)
			} else {
				logx.Warnf("desktop.statusbar_action action=pause_reminder row=%d reason=unknown", row)
			}
			return
		}
		if actionID >= desktop.StatusBarActionResumeReminderBase {
			row := actionID - desktop.StatusBarActionResumeReminderBase
			if reason, ok := c.reminderReasonByRow(row); ok {
				logx.Infof("desktop.statusbar_action action=resume_reminder row=%d reason=%s", row, reason)
				_, err := app.ResumeReminder(reason)
				c.logErr(ctx, err)
			} else {
				logx.Warnf("desktop.statusbar_action action=resume_reminder row=%d reason=unknown", row)
			}
			return
		}
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

	if overlayActive && !c.overlayFallbackNotified && app != nil {
		app.SendBreakFallbackNotification(state)
		c.overlayFallbackNotified = true
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

func buildCountdownLabel(state config.RuntimeState, language string) string {
	rows := listReminderRows(state)
	if len(rows) == 0 {
		return localizeNoRemindersLabel(language)
	}

	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, buildReminderTitle(row.choice, language, row.paused || !state.GlobalEnabled))
	}
	return strings.Join(lines, "\n")
}

func formatCountdown(sec int) string {
	if sec < 0 {
		return "off"
	}
	m := sec / 60
	s := sec % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

func buildStatusBarTitle(state config.RuntimeState) string {
	if !state.ShowTrayCountdown {
		return ""
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

	choice := selectAutoReminderChoice(state)
	if choice.reason == "" {
		return ""
	}
	return formatCountdown(choice.remaining)
}

func buildStatusBarProgress(state config.RuntimeState) float64 {
	if state.CurrentSession != nil && state.CurrentSession.Status == "resting" {
		// During break keep progress bar fixed; next countdown starts after break ends.
		return 1
	}

	if !state.GlobalEnabled {
		return 0
	}

	remaining, total := nearestCountdownRemainingAndTotal(state)
	if total <= 0 {
		return 0
	}
	progress := 1 - (float64(remaining) / float64(total))
	return clampProgress(progress)
}

func nearestCountdownRemainingAndTotal(state config.RuntimeState) (int, int) {
	choice := selectAutoReminderChoice(state)
	if choice.reason == "" {
		return 0, 0
	}
	return choice.remaining, choice.total
}

func selectAutoReminderChoice(state config.RuntimeState) autoReminderChoice {
	choices := listAutoReminderChoices(state)
	if len(choices) == 0 {
		return autoReminderChoice{}
	}
	return choices[0]
}

func listAutoReminderChoices(state config.RuntimeState) []autoReminderChoice {
	choices := make([]autoReminderChoice, 0, len(state.Reminders))
	for _, reminder := range state.Reminders {
		if !reminder.Enabled || reminder.Paused || reminder.NextInSec < 0 || !isRestRuntimeReminder(reminder) {
			continue
		}
		choices = append(choices, autoReminderChoice{
			reason:    reminder.ID,
			name:      strings.TrimSpace(reminder.Name),
			remaining: reminder.NextInSec,
			total:     reminder.IntervalSec,
		})
	}
	sort.SliceStable(choices, func(i, j int) bool {
		if choices[i].remaining == choices[j].remaining {
			return choices[i].reason < choices[j].reason
		}
		return choices[i].remaining < choices[j].remaining
	})
	return choices
}

func buildReminderTitle(choice autoReminderChoice, language string, paused bool) string {
	reasonText := reminderDisplayName(choice, language)
	if paused {
		if language == config.UILanguageZhCN {
			return fmt.Sprintf("%s - 已暂停", reasonText)
		}
		if strings.TrimSpace(choice.name) == "" {
			return fmt.Sprintf("%s - Paused", titleCaseASCII(reasonText))
		}
		return fmt.Sprintf("%s - Paused", reasonText)
	}
	countdownText := formatCountdown(choice.remaining)
	return fmt.Sprintf("%s - %s", reasonText, countdownText)
}

func reminderDisplayName(choice autoReminderChoice, language string) string {
	name := strings.TrimSpace(choice.name)
	if name != "" {
		return name
	}
	return localizeReminderReason(choice.reason, language)
}

func isRestRuntimeReminder(reminder config.ReminderRuntime) bool {
	return strings.ToLower(strings.TrimSpace(reminder.ReminderType)) != "notify"
}

func buildRemindersPayload(state config.RuntimeState, language string) (string, []string) {
	rows := listReminderRows(state)
	if len(rows) == 0 {
		return "", nil
	}

	items := make([]reminderStatusView, 0, len(rows))
	order := make([]string, 0, len(rows))
	for _, row := range rows {
		paused := row.paused || !state.GlobalEnabled
		items = append(items, reminderStatusView{
			Reason: row.choice.reason,
			Paused: paused,
			Title:  buildReminderTitle(row.choice, language, paused),
			Progress: func() float64 {
				if paused {
					return 0
				}
				return reminderProgress(row.choice)
			}(),
		})
		order = append(order, row.choice.reason)
	}

	encoded, err := json.Marshal(items)
	if err != nil {
		return "", order
	}
	return string(encoded), order
}

type reminderRow struct {
	choice autoReminderChoice
	paused bool
}

func listReminderRows(state config.RuntimeState) []reminderRow {
	rows := make([]reminderRow, 0, len(state.Reminders))
	for _, reminder := range state.Reminders {
		if !reminder.Enabled || !isRestRuntimeReminder(reminder) {
			continue
		}
		rows = append(rows, reminderRow{
			choice: autoReminderChoice{
				reason:    reminder.ID,
				name:      strings.TrimSpace(reminder.Name),
				remaining: maxInt(reminder.NextInSec, 0),
				total:     reminder.IntervalSec,
			},
			paused: reminder.Paused,
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].paused != rows[j].paused {
			return !rows[i].paused
		}
		if rows[i].paused {
			return rows[i].choice.reason < rows[j].choice.reason
		}
		if rows[i].choice.remaining == rows[j].choice.remaining {
			return rows[i].choice.reason < rows[j].choice.reason
		}
		return rows[i].choice.remaining < rows[j].choice.remaining
	})
	return rows
}

func (c *wailsDesktopController) setReminderActionOrder(order []string) {
	c.reminderOrderMu.Lock()
	defer c.reminderOrderMu.Unlock()
	c.reminderActionOrder = append([]string(nil), order...)
}

func (c *wailsDesktopController) reminderReasonByRow(row int) (string, bool) {
	if row < 0 {
		return "", false
	}
	c.reminderOrderMu.RLock()
	defer c.reminderOrderMu.RUnlock()
	if row >= len(c.reminderActionOrder) {
		return "", false
	}
	return c.reminderActionOrder[row], true
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func reminderProgress(choice autoReminderChoice) float64 {
	if choice.total <= 0 {
		return 0
	}
	progress := 1 - (float64(choice.remaining) / float64(choice.total))
	return clampProgress(progress)
}

func titleCaseASCII(text string) string {
	if text == "" {
		return ""
	}
	runes := []rune(text)
	first := runes[0]
	if first >= 'a' && first <= 'z' {
		runes[0] = first - ('a' - 'A')
	}
	return string(runes)
}

func localizeNoRemindersLabel(language string) string {
	if language == config.UILanguageZhCN {
		return "暂无提醒"
	}
	return "No reminders"
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
