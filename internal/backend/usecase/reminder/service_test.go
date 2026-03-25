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
	cloned := make([]reminderdomain.Reminder, 0, len(s.listItems))
	cloned = append(cloned, s.listItems...)
	return cloned, nil
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

type reminderRuntimeSinkStub struct {
	calls     int
	snapshots [][]reminderdomain.Reminder
}

func (s *reminderRuntimeSinkStub) ApplyReminderSnapshot(_ context.Context, reminders []reminderdomain.Reminder) error {
	s.calls++
	cloned := make([]reminderdomain.Reminder, 0, len(reminders))
	cloned = append(cloned, reminders...)
	s.snapshots = append(s.snapshots, cloned)
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

func TestCreateAppliesRuntimeSnapshot(t *testing.T) {
	restType := "rest"
	repo := &reminderRepoStub{
		listItems: []reminderdomain.Reminder{
			{ID: 1, Name: "Eye", Enabled: true, IntervalSec: 1200, BreakSec: 20, ReminderType: "rest"},
		},
	}
	sink := &reminderRuntimeSinkStub{}
	svc, err := NewService(repo)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	svc.SetRuntimeSink(sink)

	_, err = svc.Create(context.Background(), reminderdomain.CreateInput{
		Name:         "Eye",
		IntervalSec:  1200,
		BreakSec:     20,
		ReminderType: &restType,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if sink.calls != 1 {
		t.Fatalf("expected sink calls=1, got %d", sink.calls)
	}
	if len(sink.snapshots[0]) != 1 || sink.snapshots[0][0].ID != 1 {
		t.Fatalf("expected snapshot with reminder id=1, got %+v", sink.snapshots[0])
	}
}

func TestEnsureDefaultsTriggersSync(t *testing.T) {
	restType := "rest"
	repo := &reminderRepoStub{
		listItems: []reminderdomain.Reminder{
			{ID: 1, Name: "Eye", Enabled: true, IntervalSec: 1200, BreakSec: 20, ReminderType: "rest"},
		},
	}
	sink := &reminderRuntimeSinkStub{}
	svc, err := NewService(repo)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	svc.SetRuntimeSink(sink)

	if err := svc.EnsureDefaults(context.Background(), []reminderdomain.CreateInput{
		{Name: "Eye", IntervalSec: 1200, BreakSec: 20, ReminderType: &restType},
	}); err != nil {
		t.Fatalf("EnsureDefaults() error = %v", err)
	}
	if sink.calls != 1 {
		t.Fatalf("expected sink calls=1 after ensure defaults, got %d", sink.calls)
	}
}
