package reminder

import (
	"context"
	"errors"
	"strings"

	reminderdomain "pause/internal/backend/domain/reminder"
	"pause/internal/backend/ports"
)

type Service struct {
	repo ports.ReminderRepository
	sink ports.ReminderRuntimeSink
}

func NewService(repo ports.ReminderRepository, sink ports.ReminderRuntimeSink) (*Service, error) {
	if repo == nil {
		return nil, errors.New("reminder repository is required")
	}
	return &Service{repo: repo, sink: sink}, nil
}

func (s *Service) Sync(ctx context.Context) error {
	reminders, err := s.List(ctx)
	if err != nil {
		return err
	}
	return s.applyRuntimeSnapshot(ctx, reminders)
}

func (s *Service) List(ctx context.Context) ([]reminderdomain.Reminder, error) {
	items, err := s.repo.ListReminders(normalizeContext(ctx))
	if err != nil {
		return nil, err
	}
	return normalizeReminders(items), nil
}

func (s *Service) Create(ctx context.Context, input reminderdomain.CreateInput) ([]reminderdomain.Reminder, error) {
	if input.ReminderType == nil {
		return nil, errors.New("reminder reminderType is required")
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	normalized := input
	normalized.Enabled = &enabled

	if _, err := s.repo.CreateReminder(normalizeContext(ctx), normalized); err != nil {
		return nil, err
	}
	reminders, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.applyRuntimeSnapshot(ctx, reminders); err != nil {
		return nil, err
	}
	return reminders, nil
}

func (s *Service) EnsureDefaults(ctx context.Context, inputs []reminderdomain.CreateInput) error {
	for _, input := range inputs {
		if input.ReminderType == nil {
			return errors.New("reminder reminderType is required")
		}

		enabled := true
		if input.Enabled != nil {
			enabled = *input.Enabled
		}
		normalized := input
		normalized.Enabled = &enabled

		if _, err := s.repo.CreateReminder(normalizeContext(ctx), normalized); err != nil && !errors.Is(err, reminderdomain.ErrAlreadyExists) {
			return err
		}
	}
	return s.Sync(ctx)
}

func (s *Service) Update(ctx context.Context, patch reminderdomain.Patch) ([]reminderdomain.Reminder, error) {
	if err := s.repo.UpdateReminder(normalizeContext(ctx), patch); err != nil {
		return nil, err
	}
	reminders, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.applyRuntimeSnapshot(ctx, reminders); err != nil {
		return nil, err
	}
	return reminders, nil
}

func (s *Service) Delete(ctx context.Context, reminderID int64) ([]reminderdomain.Reminder, error) {
	if reminderID <= 0 {
		return nil, errors.New("reminder id is required")
	}
	if err := s.repo.DeleteReminder(normalizeContext(ctx), reminderID); err != nil {
		return nil, err
	}
	reminders, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.applyRuntimeSnapshot(ctx, reminders); err != nil {
		return nil, err
	}
	return reminders, nil
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

func normalizeReminders(reminders []reminderdomain.Reminder) []reminderdomain.Reminder {
	if len(reminders) == 0 {
		return nil
	}
	result := make([]reminderdomain.Reminder, 0, len(reminders))
	for _, item := range reminders {
		if item.ID <= 0 {
			continue
		}
		result = append(result, reminderdomain.Reminder{
			ID:           item.ID,
			Name:         strings.TrimSpace(item.Name),
			Enabled:      item.Enabled,
			IntervalSec:  item.IntervalSec,
			BreakSec:     item.BreakSec,
			ReminderType: strings.TrimSpace(item.ReminderType),
		})
	}
	if len(result) == 0 {
		return nil
	}
	cloned := make([]reminderdomain.Reminder, 0, len(result))
	cloned = append(cloned, result...)
	return cloned
}

func (s *Service) applyRuntimeSnapshot(ctx context.Context, reminders []reminderdomain.Reminder) error {
	if s == nil || s.sink == nil {
		return nil
	}
	return s.sink.ApplyReminderSnapshot(normalizeContext(ctx), reminders)
}
