package app

import (
	"fmt"
	"strings"
	"time"

	"pause/internal/backend/runtime/state"
	"pause/internal/logx"
	"pause/internal/platform"
)

func (a *App) SendBreakFallbackNotification(state state.RuntimeState) {
	if a.notifier == nil {
		logx.Warnf("overlay.fallback_notification_skipped reason=no_notifier")
		return
	}
	reasons := "none"
	if state.CurrentSession != nil {
		reasons = joinReasons(state.CurrentSession.Reasons)
	}
	body := buildBreakNotificationBody(state)
	notifier := a.notifier
	go func(reasonKey string, n platform.Notifier, message string) {
		if err := n.ShowReminder("Time to rest", message); err != nil {
			logx.Warnf("overlay.fallback_notification_err err=%v", err)
			return
		}
		logx.Infof("overlay.fallback_notification_sent reasons=%s", reasonKey)
	}(reasons, notifier, body)
}

func buildBreakNotificationBody(state state.RuntimeState) string {
	if state.CurrentSession == nil {
		return "Break started"
	}

	namesByReason := map[int64]string{}
	for _, reminder := range state.Reminders {
		id := reminder.ID
		name := strings.TrimSpace(reminder.Name)
		if id <= 0 || name == "" {
			continue
		}
		namesByReason[id] = name
	}

	parts := make([]string, 0, len(state.CurrentSession.Reasons))
	for _, reason := range state.CurrentSession.Reasons {
		if reason <= 0 {
			continue
		}
		if name, ok := namesByReason[reason]; ok && name != "" {
			parts = append(parts, name)
		}
	}

	label := strings.Join(parts, " + ")

	if state.CurrentSession.RemainingSec > 0 {
		duration := (time.Duration(state.CurrentSession.RemainingSec) * time.Second).String()
		return fmt.Sprintf("%s break for %s", label, duration)
	}
	return label + " break started"
}

func joinReasons(reasons []int64) string {
	if len(reasons) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		parts = append(parts, fmt.Sprintf("%d", reason))
	}
	return strings.Join(parts, "+")
}
