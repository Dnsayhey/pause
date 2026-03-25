package historyadapter

import (
	"context"
	"errors"

	reminderdomain "pause/internal/backend/domain/reminder"
	"pause/internal/backend/ports"
	historydb "pause/internal/backend/storage/historydb"
)

var errHistoryStoreUnavailable = errors.New("history store unavailable")

type ReminderRepository struct {
	store *historydb.Store
}

var _ ports.ReminderRepository = (*ReminderRepository)(nil)

func NewReminderRepository(store *historydb.Store) *ReminderRepository {
	return &ReminderRepository{store: store}
}

func (r *ReminderRepository) ListReminders(ctx context.Context) ([]reminderdomain.Reminder, error) {
	if err := r.ensureStore(); err != nil {
		return nil, err
	}
	items, err := r.store.ListReminders(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]reminderdomain.Reminder, 0, len(items))
	for _, item := range items {
		result = append(result, reminderdomain.Reminder{
			ID:           item.ID,
			Name:         item.Name,
			Enabled:      item.Enabled,
			IntervalSec:  item.IntervalSec,
			BreakSec:     item.BreakSec,
			ReminderType: item.ReminderType,
		})
	}
	return result, nil
}

func (r *ReminderRepository) CreateReminder(ctx context.Context, input reminderdomain.CreateInput) (int64, error) {
	if err := r.ensureStore(); err != nil {
		return 0, err
	}

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	reminderType := ""
	if input.ReminderType != nil {
		reminderType = *input.ReminderType
	}

	id, err := r.store.CreateReminder(ctx, historydb.Reminder{
		Name:         input.Name,
		Enabled:      enabled,
		IntervalSec:  input.IntervalSec,
		BreakSec:     input.BreakSec,
		ReminderType: reminderType,
	})
	if err != nil {
		if errors.Is(err, historydb.ErrReminderAlreadyExists) {
			return 0, reminderdomain.ErrAlreadyExists
		}
		return 0, err
	}
	return id, nil
}

func (r *ReminderRepository) UpdateReminder(ctx context.Context, patch reminderdomain.Patch) error {
	if err := r.ensureStore(); err != nil {
		return err
	}

	err := r.store.UpdateReminder(ctx, patch.ID, historydb.ReminderPatch{
		Name:         patch.Name,
		Enabled:      patch.Enabled,
		IntervalSec:  patch.IntervalSec,
		BreakSec:     patch.BreakSec,
		ReminderType: patch.ReminderType,
	})
	if err != nil {
		if errors.Is(err, historydb.ErrReminderNotFound) {
			return reminderdomain.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *ReminderRepository) DeleteReminder(ctx context.Context, reminderID int64) error {
	if err := r.ensureStore(); err != nil {
		return err
	}
	err := r.store.DeleteReminder(ctx, reminderID)
	if err != nil {
		if errors.Is(err, historydb.ErrReminderNotFound) {
			return reminderdomain.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *ReminderRepository) ensureStore() error {
	if r == nil || r.store == nil {
		return errHistoryStoreUnavailable
	}
	return nil
}
