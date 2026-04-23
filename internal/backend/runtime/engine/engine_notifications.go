package engine

import (
	"context"
	"strconv"
	"strings"

	"pause/internal/backend/domain/reminder"
	"pause/internal/backend/domain/settings"
	"pause/internal/backend/ports"
	"pause/internal/logx"
)

func (e *Engine) notifyRemindersLocked(reminderIDs []int64, language string) {
	if len(reminderIDs) == 0 || e.notifier == nil {
		return
	}
	names := make([]string, 0, len(reminderIDs))
	byID := make(map[int64]reminder.Reminder, len(e.reminders))
	for _, reminder := range e.reminders {
		byID[reminder.ID] = reminder
	}
	for _, id := range reminderIDs {
		reminder, ok := byID[id]
		if !ok {
			continue
		}
		name := strings.TrimSpace(reminder.Name)
		if name == "" {
			name = strconv.FormatInt(reminder.ID, 10)
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return
	}

	title := "Reminder"
	body := strings.Join(names, " · ")
	if language == settings.UILanguageZhCN {
		title = "提醒"
	}
	notifier := e.notifier
	runCtx := e.runCtx
	if runCtx == nil {
		runCtx = context.Background()
	}
	keyParts := make([]string, 0, len(reminderIDs))
	for _, id := range reminderIDs {
		keyParts = append(keyParts, strconv.FormatInt(id, 10))
	}
	reminderKey := strings.Join(keyParts, "+")
	e.backgroundTasks.Add(1)
	go func(ctx context.Context, n ports.Notifier, t string, b string, key string) {
		defer e.backgroundTasks.Done()
		if err := n.ShowReminder(ctx, t, b); err != nil {
			logx.Warnf("reminder.notification_err reminders=%s err=%v", key, err)
			return
		}
		logx.Infof("reminder.notification_sent reminders=%s", key)
	}(runCtx, notifier, title, body, reminderKey)
}
