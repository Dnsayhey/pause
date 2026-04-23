package engine

import (
	"context"
	"errors"
	"time"

	"pause/internal/backend/domain/reminder"
	"pause/internal/backend/runtime/state"
	"pause/internal/logx"
)

func (e *Engine) SetReminderConfigs(reminders []reminder.Reminder) []reminder.Reminder {
	e.mu.Lock()
	defer e.mu.Unlock()

	next := cloneReminderConfigs(reminders)
	prev := cloneReminderConfigs(e.reminders)
	e.reminders = next
	e.applyReminderConfigPatchLocked(prev, next)
	logx.Infof("reminders.synced count=%d", len(e.reminders))
	return cloneReminderConfigs(next)
}

func (e *Engine) ApplyReminderSnapshot(ctx context.Context, reminders []reminder.Reminder) error {
	_ = ctx
	e.SetReminderConfigs(reminders)
	return nil
}

func (e *Engine) PauseReminder(reminderID int64, now time.Time) (state.RuntimeState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if reminderID <= 0 {
		return state.RuntimeState{}, errors.New("invalid reminder reason")
	}
	if _, ok := findReminderByID(e.reminders, reminderID); !ok {
		return state.RuntimeState{}, errors.New("unknown reminder reason")
	}
	wasPaused := e.pausedReminder[reminderID]
	e.pausedReminder[reminderID] = true
	logx.Infof("reminder.pause reason=%d already_paused=%t", reminderID, wasPaused)

	settings := e.store.Get()
	return e.runtimeStateLocked(now, settings), nil
}

func (e *Engine) ResumeReminder(reminderID int64, now time.Time) (state.RuntimeState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if reminderID <= 0 {
		return state.RuntimeState{}, errors.New("invalid reminder reason")
	}
	wasPaused := e.pausedReminder[reminderID]
	delete(e.pausedReminder, reminderID)
	logx.Infof("reminder.resume reason=%d was_paused=%t", reminderID, wasPaused)

	settings := e.store.Get()
	return e.runtimeStateLocked(now, settings), nil
}

func (e *Engine) applyReminderConfigPatchLocked(prev, next []reminder.Reminder) {
	prevByID := map[int64]reminder.Reminder{}
	for _, reminder := range prev {
		prevByID[reminder.ID] = reminder
	}
	nextByID := map[int64]reminder.Reminder{}
	for _, reminder := range next {
		nextByID[reminder.ID] = reminder
	}

	ids := map[int64]struct{}{}
	for id := range prevByID {
		ids[id] = struct{}{}
	}
	for id := range nextByID {
		ids[id] = struct{}{}
	}

	for id := range ids {
		p, hasPrev := prevByID[id]
		n, hasNext := nextByID[id]
		changed := !hasPrev || !hasNext ||
			p.Enabled != n.Enabled ||
			p.IntervalSec != n.IntervalSec
		if !changed {
			continue
		}
		e.scheduler.ResetByID(id)
		delete(e.pausedReminder, id)
	}
}

func (e *Engine) effectiveReminderConfigsLocked(reminders []reminder.Reminder) []reminder.Reminder {
	updated := make([]reminder.Reminder, 0, len(reminders))
	for _, reminder := range reminders {
		next := reminder
		if next.Enabled && e.pausedReminder[next.ID] {
			next.Enabled = false
		}
		updated = append(updated, next)
	}
	return updated
}
