package reminder

import (
	"context"
	"errors"
	"testing"

	reminderdomain "pause/internal/backend/domain/reminder"
)

type reminderRepoStub struct {
	createdInputs []reminderdomain.CreateInput
	createErrBy   map[string]error
}

func (s *reminderRepoStub) ListReminders(_ context.Context) ([]reminderdomain.Reminder, error) {
	return nil, nil
}

func (s *reminderRepoStub) CreateReminder(_ context.Context, input reminderdomain.CreateInput) (int64, error) {
	s.createdInputs = append(s.createdInputs, input)
	if err, ok := s.createErrBy[input.Name]; ok {
		return 0, err
	}
	return int64(len(s.createdInputs)), nil
}

func (s *reminderRepoStub) UpdateReminder(_ context.Context, _ reminderdomain.Patch) error {
	return nil
}

func (s *reminderRepoStub) DeleteReminder(_ context.Context, _ int64) error {
	return nil
}

func TestEnsureDefaultsIgnoresAlreadyExists(t *testing.T) {
	restType := "rest"
	notifyType := "notify"
	repo := &reminderRepoStub{
		createErrBy: map[string]error{
			"Eye": reminderdomain.ErrAlreadyExists,
		},
	}
	svc, err := NewService(repo)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	err = svc.EnsureDefaults(context.Background(), []reminderdomain.CreateInput{
		{Name: "Eye", IntervalSec: 1200, BreakSec: 20, ReminderType: &restType},
		{Name: "Hydrate", IntervalSec: 2700, BreakSec: 1, ReminderType: &notifyType},
	})
	if err != nil {
		t.Fatalf("EnsureDefaults() error = %v", err)
	}
	if got, want := len(repo.createdInputs), 2; got != want {
		t.Fatalf("expected %d create attempts, got %d", want, got)
	}
}

func TestEnsureDefaultsSetsEnabledTrueByDefault(t *testing.T) {
	restType := "rest"
	repo := &reminderRepoStub{}
	svc, err := NewService(repo)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if err := svc.EnsureDefaults(context.Background(), []reminderdomain.CreateInput{
		{Name: "Eye", IntervalSec: 1200, BreakSec: 20, ReminderType: &restType},
	}); err != nil {
		t.Fatalf("EnsureDefaults() error = %v", err)
	}

	if len(repo.createdInputs) != 1 {
		t.Fatalf("expected one created input, got %d", len(repo.createdInputs))
	}
	enabled := repo.createdInputs[0].Enabled
	if enabled == nil || !*enabled {
		t.Fatalf("expected default enabled=true")
	}
}

func TestEnsureDefaultsFailsOnUnexpectedError(t *testing.T) {
	restType := "rest"
	wantErr := errors.New("db down")
	repo := &reminderRepoStub{
		createErrBy: map[string]error{
			"Eye": wantErr,
		},
	}
	svc, err := NewService(repo)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	err = svc.EnsureDefaults(context.Background(), []reminderdomain.CreateInput{
		{Name: "Eye", IntervalSec: 1200, BreakSec: 20, ReminderType: &restType},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}
