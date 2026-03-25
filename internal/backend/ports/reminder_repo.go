package ports

import (
	"context"

	reminderdomain "pause/internal/backend/domain/reminder"
)

type ReminderRepository interface {
	ListReminders(ctx context.Context) ([]reminderdomain.Reminder, error)
	CreateReminder(ctx context.Context, input reminderdomain.CreateInput) (int64, error)
	UpdateReminder(ctx context.Context, patch reminderdomain.Patch) error
	DeleteReminder(ctx context.Context, reminderID int64) error
}
