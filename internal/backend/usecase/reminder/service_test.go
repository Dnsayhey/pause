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
	listItems     []reminderdomain.Reminder
}

func (s *reminderRepoStub) ListReminders(_ context.Context) ([]reminderdomain.Reminder, error) {
	if len(s.listItems) == 0 {
		return nil, nil
	}
	out := make([]reminderdomain.Reminder, 0, len(s.listItems))
	out = append(out, s.listItems...)
	return out, nil
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
func (s *reminderRepoStub) DeleteReminder(_ context.Context, _ int64) error { return nil }

type runtimeSinkStub struct {
	calls int
}

func (s *runtimeSinkStub) ApplyReminderSnapshot(_ context.Context, _ []reminderdomain.Reminder) error {
	s.calls++
	return nil
}

func TestReminderService_EnsureDefaultsIgnoresAlreadyExists(t *testing.T) {
	rest := "rest"
	notify := "notify"
	repo := &reminderRepoStub{createErrBy: map[string]error{"Eye": reminderdomain.ErrAlreadyExists}}
	svc, err := NewService(repo, nil)
	if err != nil {
		t.Fatalf("NewService() err=%v", err)
	}

	err = svc.EnsureDefaults(context.Background(), []reminderdomain.CreateInput{
		{Name: "Eye", IntervalSec: 1200, BreakSec: 20, ReminderType: &rest},
		{Name: "Hydrate", IntervalSec: 2700, BreakSec: 1, ReminderType: &notify},
	})
	if err != nil {
		t.Fatalf("EnsureDefaults() err=%v", err)
	}
	if len(repo.createdInputs) != 2 {
		t.Fatalf("create attempts mismatch: got=%d want=2", len(repo.createdInputs))
	}
}

func TestReminderService_CreateSetsEnabledTrueAndSyncs(t *testing.T) {
	rest := "rest"
	repo := &reminderRepoStub{listItems: []reminderdomain.Reminder{{ID: 1, Name: "Eye", Enabled: true, IntervalSec: 1200, BreakSec: 20, ReminderType: "rest"}}}
	sink := &runtimeSinkStub{}
	svc, err := NewService(repo, sink)
	if err != nil {
		t.Fatalf("NewService() err=%v", err)
	}

	_, err = svc.Create(context.Background(), reminderdomain.CreateInput{Name: "Eye", IntervalSec: 1200, BreakSec: 20, ReminderType: &rest})
	if err != nil {
		t.Fatalf("Create() err=%v", err)
	}
	if len(repo.createdInputs) != 1 {
		t.Fatalf("expected one create call")
	}
	if repo.createdInputs[0].Enabled == nil || !*repo.createdInputs[0].Enabled {
		t.Fatalf("expected default enabled=true")
	}
	if sink.calls != 1 {
		t.Fatalf("expected runtime sink sync once, got=%d", sink.calls)
	}
}

func TestReminderService_EnsureDefaultsUnexpectedError(t *testing.T) {
	rest := "rest"
	wantErr := errors.New("db down")
	repo := &reminderRepoStub{createErrBy: map[string]error{"Eye": wantErr}}
	svc, err := NewService(repo, nil)
	if err != nil {
		t.Fatalf("NewService() err=%v", err)
	}

	err = svc.EnsureDefaults(context.Background(), []reminderdomain.CreateInput{{Name: "Eye", IntervalSec: 1200, BreakSec: 20, ReminderType: &rest}})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error mismatch got=%v want=%v", err, wantErr)
	}
}

func TestReminderService_CreateRejectsInvalidReminder(t *testing.T) {
	rest := "rest"
	svc, err := NewService(&reminderRepoStub{}, nil)
	if err != nil {
		t.Fatalf("NewService() err=%v", err)
	}

	_, err = svc.Create(context.Background(), reminderdomain.CreateInput{
		Name:         "Eye",
		IntervalSec:  0,
		BreakSec:     20,
		ReminderType: &rest,
	})
	if !errors.Is(err, reminderdomain.ErrIntervalRange) {
		t.Fatalf("Create() err=%v want=%v", err, reminderdomain.ErrIntervalRange)
	}
}

func TestReminderService_UpdateRejectsInvalidPatch(t *testing.T) {
	svc, err := NewService(&reminderRepoStub{}, nil)
	if err != nil {
		t.Fatalf("NewService() err=%v", err)
	}
	name := " "
	if _, err := svc.Update(context.Background(), reminderdomain.Patch{ID: 1, Name: &name}); !errors.Is(err, reminderdomain.ErrNameRequired) {
		t.Fatalf("Update() err=%v want=%v", err, reminderdomain.ErrNameRequired)
	}
}
