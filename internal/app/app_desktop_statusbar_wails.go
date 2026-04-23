//go:build wails

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	reminderdomain "pause/internal/backend/domain/reminder"
	"pause/internal/backend/domain/settings"
	"pause/internal/backend/runtime/state"
	"pause/internal/desktop"
	"pause/internal/logx"
)

type autoReminderChoice struct {
	reason    int64
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

type reminderRow struct {
	choice autoReminderChoice
	paused bool
}

func (c *wailsDesktopController) syncStatusBarWithLock(state state.RuntimeState, settings settings.Settings) {
	c.statusBarSyncMu.Lock()
	defer c.statusBarSyncMu.Unlock()
	c.syncStatusBar(state, settings)
}

func (c *wailsDesktopController) syncStatusBar(state state.RuntimeState, settings settings.Settings) {
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
			// Keep native status-item click callback non-blocking.
			// If we synchronously refresh here, we can deadlock with the runtime loop:
			// runtime loop holds statusBarSyncMu while waiting for main-thread UI update,
			// and this callback runs on main thread waiting for statusBarSyncMu.
			go func() {
				settings := app.engine.GetSettings()
				state := app.engine.GetRuntimeState(time.Now())
				c.syncStatusBarWithLock(state, settings)
			}()
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
				logx.Infof("desktop.statusbar_action action=pause_reminder row=%d reason=%d", row, reason)
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
				logx.Infof("desktop.statusbar_action action=resume_reminder row=%d reason=%d", row, reason)
				_, err := app.ResumeReminder(reason)
				c.logErr(ctx, err)
			} else {
				logx.Warnf("desktop.statusbar_action action=resume_reminder row=%d reason=unknown", row)
			}
			return
		}
	}
}

func buildPauseLabel(state state.RuntimeState, language string) string {
	if !state.GlobalEnabled {
		if language == settings.UILanguageZhCN {
			return "状态：已关闭"
		}
		return "Status: disabled"
	}
	if state.CurrentSession != nil && state.CurrentSession.Status == "resting" {
		if language == settings.UILanguageZhCN {
			return "状态：休息中"
		}
		return "Status: on break"
	}
	if language == settings.UILanguageZhCN {
		return "状态：运行中"
	}
	return "Status: running"
}

func buildCountdownLabel(state state.RuntimeState, language string) string {
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

func buildStatusBarTitle(state state.RuntimeState) string {
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
	if choice.reason <= 0 {
		return ""
	}
	return formatCountdown(choice.remaining)
}

func buildStatusBarProgress(state state.RuntimeState) float64 {
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

func nearestCountdownRemainingAndTotal(state state.RuntimeState) (int, int) {
	choice := selectAutoReminderChoice(state)
	if choice.reason <= 0 {
		return 0, 0
	}
	return choice.remaining, choice.total
}

func selectAutoReminderChoice(state state.RuntimeState) autoReminderChoice {
	choices := listAutoReminderChoices(state)
	if len(choices) == 0 {
		return autoReminderChoice{}
	}
	return choices[0]
}

func listAutoReminderChoices(state state.RuntimeState) []autoReminderChoice {
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
		if language == settings.UILanguageZhCN {
			return fmt.Sprintf("%s - 已暂停", reasonText)
		}
		return fmt.Sprintf("%s - Paused", reasonText)
	}
	countdownText := formatCountdown(choice.remaining)
	return fmt.Sprintf("%s - %s", reasonText, countdownText)
}

func reminderDisplayName(choice autoReminderChoice, language string) string {
	_ = language
	return strings.TrimSpace(choice.name)
}

func isRestRuntimeReminder(reminder state.ReminderRuntime) bool {
	return reminderdomain.IsRestReminderType(reminder.ReminderType)
}

func buildRemindersPayload(state state.RuntimeState, language string) (string, []int64) {
	rows := listReminderRows(state)
	if len(rows) == 0 {
		return "", nil
	}

	items := make([]reminderStatusView, 0, len(rows))
	order := make([]int64, 0, len(rows))
	for _, row := range rows {
		paused := row.paused || !state.GlobalEnabled
		items = append(items, reminderStatusView{
			Reason: strconv.FormatInt(row.choice.reason, 10),
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

func listReminderRows(state state.RuntimeState) []reminderRow {
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

func (c *wailsDesktopController) setReminderActionOrder(order []int64) {
	c.reminderOrderMu.Lock()
	defer c.reminderOrderMu.Unlock()
	c.reminderActionOrder = append([]int64(nil), order...)
}

func (c *wailsDesktopController) reminderReasonByRow(row int) (int64, bool) {
	if row < 0 {
		return 0, false
	}
	c.reminderOrderMu.RLock()
	defer c.reminderOrderMu.RUnlock()
	if row >= len(c.reminderActionOrder) {
		return 0, false
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

func localizeNoRemindersLabel(language string) string {
	if language == settings.UILanguageZhCN {
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
