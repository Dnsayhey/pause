package app

import (
	"context"
	"path/filepath"
	"testing"

	"pause/internal/core/config"
	"pause/internal/core/history"
	"pause/internal/core/service"
)

func newTestAppWithHistory(t *testing.T) *App {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "settings.json")
	store, err := config.NewStore(configPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	historyPath := filepath.Join(t.TempDir(), "history.db")
	historyStore, err := history.OpenStore(context.Background(), historyPath)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	t.Cleanup(func() { _ = historyStore.Close() })

	engine := service.NewEngine(store, nil, nil, nil, nil, historyStore)
	defs, err := historyStore.ListReminders(context.Background())
	if err != nil {
		t.Fatalf("ListReminders() error = %v", err)
	}
	engine.SetReminderConfigs(historyDefsToConfig(defs))

	return &App{
		ctx:     context.Background(),
		engine:  engine,
		history: historyStore,
	}
}

func TestAppCreateReminderRejectsMissingType(t *testing.T) {
	app := newTestAppWithHistory(t)

	_, err := app.CreateReminder(config.ReminderCreateInput{
		Name:        "Focus",
		IntervalSec: 1500,
		BreakSec:    30,
	})
	if err == nil {
		t.Fatalf("expected missing reminder type to fail")
	}
}

func TestAppReminderCRUDLifecycle(t *testing.T) {
	app := newTestAppWithHistory(t)

	restType := "rest"
	created, err := app.CreateReminder(config.ReminderCreateInput{
		Name:         "Focus",
		IntervalSec:  1500,
		BreakSec:     30,
		ReminderType: &restType,
	})
	if err != nil {
		t.Fatalf("CreateReminder() error = %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("expected 1 reminder after create, got %d", len(created))
	}
	id := created[0].ID
	if id <= 0 {
		t.Fatalf("expected created reminder id > 0, got %d", id)
	}

	newName := "Deep Focus"
	enabled := false
	updated, err := app.UpdateReminder(config.ReminderPatch{
		ID:      id,
		Name:    &newName,
		Enabled: &enabled,
	})
	if err != nil {
		t.Fatalf("UpdateReminder() error = %v", err)
	}
	if len(updated) != 1 {
		t.Fatalf("expected 1 reminder after update, got %d", len(updated))
	}
	if updated[0].Name != newName || updated[0].Enabled != enabled {
		t.Fatalf("expected updated reminder values, got name=%q enabled=%t", updated[0].Name, updated[0].Enabled)
	}

	afterDelete, err := app.DeleteReminder(id)
	if err != nil {
		t.Fatalf("DeleteReminder() error = %v", err)
	}
	if len(afterDelete) != 0 {
		t.Fatalf("expected 0 reminders after delete, got %d", len(afterDelete))
	}
}
