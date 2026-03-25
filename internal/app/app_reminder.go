package app

import (
	"context"
	"errors"
	"strings"

	reminderdomain "pause/internal/backend/domain/reminder"
	"pause/internal/core/history"
	"pause/internal/core/reminder"
	"pause/internal/logx"
)

func (a *App) GetReminders() ([]reminder.ReminderConfig, error) {
	if a == nil || a.reminders == nil {
		return nil, errors.New("reminder service unavailable")
	}
	defs, err := a.reminders.List(appContextOrBackground(a.ctx))
	if err != nil {
		return nil, err
	}
	return reminderDefsToConfig(defs), nil
}

func (a *App) UpdateReminder(patch reminder.ReminderPatch) ([]reminder.ReminderConfig, error) {
	if a == nil || a.reminders == nil {
		return nil, errors.New("reminder service unavailable")
	}
	ctx := appContextOrBackground(a.ctx)
	defs, err := a.reminders.Update(ctx, reminderdomain.Patch{
		ID:           patch.ID,
		Name:         patch.Name,
		Enabled:      patch.Enabled,
		IntervalSec:  patch.IntervalSec,
		BreakSec:     patch.BreakSec,
		ReminderType: patch.ReminderType,
	})
	if err != nil {
		logx.Warnf("app.update_reminder_err stage=usecase_update patch_id=%d err=%v", patch.ID, err)
		return nil, err
	}
	reminders := reminderDefsToConfig(defs)
	a.engine.SetReminderConfigs(reminders)
	logx.Infof("app.reminder_updated id=%d count=%d", patch.ID, len(reminders))
	return reminders, nil
}

func (a *App) CreateReminder(input reminder.ReminderCreateInput) ([]reminder.ReminderConfig, error) {
	if a == nil || a.reminders == nil {
		return nil, errors.New("reminder service unavailable")
	}

	ctx := appContextOrBackground(a.ctx)
	defs, err := a.reminders.Create(ctx, reminderdomain.CreateInput{
		Name:         input.Name,
		IntervalSec:  input.IntervalSec,
		BreakSec:     input.BreakSec,
		Enabled:      input.Enabled,
		ReminderType: input.ReminderType,
	})
	if err != nil {
		logx.Warnf("app.create_reminder_err stage=usecase_create err=%v", err)
		return nil, err
	}
	reminders := reminderDefsToConfig(defs)
	a.engine.SetReminderConfigs(reminders)
	createdID := int64(0)
	if len(reminders) > 0 {
		createdID = reminders[len(reminders)-1].ID
	}
	logx.Infof("app.reminder_created id=%d count=%d", createdID, len(reminders))
	return reminders, nil
}

func (a *App) DeleteReminder(reminderID int64) ([]reminder.ReminderConfig, error) {
	if a == nil || a.reminders == nil {
		return nil, errors.New("reminder service unavailable")
	}
	id := reminderID
	ctx := appContextOrBackground(a.ctx)
	defs, err := a.reminders.Delete(ctx, id)
	if err != nil {
		logx.Warnf("app.delete_reminder_err stage=usecase_delete id=%d err=%v", id, err)
		return nil, err
	}
	reminders := reminderDefsToConfig(defs)
	a.engine.SetReminderConfigs(reminders)
	logx.Infof("app.reminder_deleted id=%d count=%d", id, len(reminders))
	return reminders, nil
}

func reminderDefsToConfig(defs []reminderdomain.Reminder) []reminder.ReminderConfig {
	result := make([]reminder.ReminderConfig, 0, len(defs))
	for _, def := range defs {
		id := def.ID
		if id <= 0 {
			continue
		}
		result = append(result, reminder.ReminderConfig{
			ID:           id,
			Name:         strings.TrimSpace(def.Name),
			Enabled:      def.Enabled,
			IntervalSec:  def.IntervalSec,
			BreakSec:     def.BreakSec,
			ReminderType: strings.TrimSpace(def.ReminderType),
		})
	}
	return cloneReminderConfigs(result)
}

func cloneReminderConfigs(reminders []reminder.ReminderConfig) []reminder.ReminderConfig {
	if len(reminders) == 0 {
		return nil
	}
	cloned := make([]reminder.ReminderConfig, 0, len(reminders))
	cloned = append(cloned, reminders...)
	return cloned
}

func ensureBuiltInRemindersForFirstInstall(ctx context.Context, store *history.HistoryStore, language string) error {
	if store == nil {
		return nil
	}

	eyeName, standName, waterName := localizedBuiltInReminderSeedNames(language)
	eyeEnabled := true
	standEnabled := false
	waterEnabled := false
	restType := "rest"
	notifyType := "notify"
	eyeIntervalSec := 20 * 60
	eyeBreakSec := 20
	standIntervalSec := 60 * 60
	standBreakSec := 5 * 60
	waterIntervalSec := 45 * 60
	waterBreakSec := 1
	reminders := []history.Reminder{
		{
			Name:         eyeName,
			Enabled:      eyeEnabled,
			IntervalSec:  eyeIntervalSec,
			BreakSec:     eyeBreakSec,
			ReminderType: restType,
		},
		{
			Name:         standName,
			Enabled:      standEnabled,
			IntervalSec:  standIntervalSec,
			BreakSec:     standBreakSec,
			ReminderType: restType,
		},
		{
			Name:         waterName,
			Enabled:      waterEnabled,
			IntervalSec:  waterIntervalSec,
			BreakSec:     waterBreakSec,
			ReminderType: notifyType,
		},
	}

	for _, reminder := range reminders {
		if _, err := store.CreateReminder(appContextOrBackground(ctx), reminder); err != nil && !errors.Is(err, history.ErrReminderAlreadyExists) {
			return err
		}
	}
	return nil
}
