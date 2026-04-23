package app

import (
	"context"
	"path/filepath"
	"testing"

	historyadapter "pause/internal/backend/adapters/history"
	"pause/internal/backend/bootstrap"
	"pause/internal/backend/ports"
	service "pause/internal/backend/runtime/engine"
	"pause/internal/backend/storage/historydb"
	"pause/internal/backend/storage/settingsjson"
)

func newTestApp(t *testing.T) *App {
	t.Helper()

	settingsStore, err := settingsjson.OpenStore(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("OpenStore(settings) err=%v", err)
	}
	historyStore, err := historydb.OpenStore(context.Background(), filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatalf("OpenStore(history) err=%v", err)
	}
	t.Cleanup(func() { _ = historyStore.Close() })

	container, err := bootstrap.NewContainer(historyStore, nil)
	if err != nil {
		t.Fatalf("NewContainer() err=%v", err)
	}
	engine := service.NewEngine(settingsStore, nil, nil, nil, nil, historyadapter.NewBreakRecorder(historyStore))
	defs, err := container.ReminderService.List(context.Background())
	if err != nil {
		t.Fatalf("ReminderService.List() err=%v", err)
	}
	engine.SetReminderConfigs(defs)

	return &App{
		ctx:       context.Background(),
		engine:    bootstrap.WrapEngine(engine),
		runtime:   historyStore,
		reminders: container.ReminderService,
	}
}

func TestCreateReminder_RequiresType(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.CreateReminder(ReminderCreateInput{Name: "Focus", IntervalSec: 1500, BreakSec: 30}); err == nil {
		t.Fatalf("expected missing reminderType error")
	}
}

func TestReminderCRUD(t *testing.T) {
	app := newTestApp(t)
	rest := "rest"

	created, err := app.CreateReminder(ReminderCreateInput{Name: "Focus", IntervalSec: 1500, BreakSec: 30, ReminderType: &rest})
	if err != nil {
		t.Fatalf("CreateReminder() err=%v", err)
	}
	if len(created) != 1 || created[0].ID <= 0 {
		t.Fatalf("unexpected create result: %+v", created)
	}
	id := created[0].ID

	name := "Deep Focus"
	enabled := false
	updated, err := app.UpdateReminder(ReminderPatch{ID: id, Name: &name, Enabled: &enabled})
	if err != nil {
		t.Fatalf("UpdateReminder() err=%v", err)
	}
	if len(updated) != 1 || updated[0].Name != name || updated[0].Enabled != enabled {
		t.Fatalf("unexpected update result: %+v", updated)
	}

	left, err := app.DeleteReminder(id)
	if err != nil {
		t.Fatalf("DeleteReminder() err=%v", err)
	}
	if left == nil {
		t.Fatalf("expected non-nil reminder slice after delete")
	}
	if len(left) != 0 {
		t.Fatalf("expected empty reminders after delete, got=%d", len(left))
	}
}

type notificationCapabilityProviderStub struct {
	capability      ports.NotificationCapability
	requested       bool
	requestOutcome  ports.NotificationCapability
	requestErr      error
	openSettingsErr error
}

func (s *notificationCapabilityProviderStub) GetNotificationCapability() ports.NotificationCapability {
	return s.capability
}

func (s *notificationCapabilityProviderStub) RequestNotificationPermission() (ports.NotificationCapability, error) {
	s.requested = true
	if s.requestErr != nil {
		return ports.NotificationCapability{}, s.requestErr
	}
	if s.requestOutcome.PermissionState != "" {
		s.capability = s.requestOutcome
	}
	return s.capability, nil
}

func (s *notificationCapabilityProviderStub) OpenNotificationSettings() error {
	return s.openSettingsErr
}

func TestCreateNotifyReminder_DeniedPermission(t *testing.T) {
	app := newTestApp(t)
	provider := &notificationCapabilityProviderStub{
		capability: ports.NotificationCapability{
			PermissionState: ports.NotificationPermissionDenied,
			CanRequest:      false,
			CanOpenSettings: true,
		},
	}
	app.notificationCapability = provider

	notify := "notify"
	created, err := app.CreateReminder(ReminderCreateInput{Name: "Hydrate", IntervalSec: 1500, BreakSec: 1, ReminderType: &notify})
	if err != nil {
		t.Fatalf("CreateReminder() err=%v", err)
	}
	if len(created) != 1 {
		t.Fatalf("expected one reminder, got=%d", len(created))
	}
	if provider.requested {
		t.Fatalf("create reminder should not request notification permission")
	}
}

func TestCreateNotifyReminder_DoesNotRequestPermission(t *testing.T) {
	app := newTestApp(t)
	provider := &notificationCapabilityProviderStub{
		capability: ports.NotificationCapability{
			PermissionState: ports.NotificationPermissionNotDetermined,
			CanRequest:      true,
			CanOpenSettings: true,
		},
		requestOutcome: ports.NotificationCapability{
			PermissionState: ports.NotificationPermissionAuthorized,
			CanRequest:      false,
			CanOpenSettings: true,
		},
	}
	app.notificationCapability = provider

	notify := "notify"
	created, err := app.CreateReminder(ReminderCreateInput{Name: "Hydrate", IntervalSec: 1500, BreakSec: 1, ReminderType: &notify})
	if err != nil {
		t.Fatalf("CreateReminder() err=%v", err)
	}
	if len(created) != 1 {
		t.Fatalf("expected one reminder, got=%d", len(created))
	}
	if provider.requested {
		t.Fatalf("create reminder should not request notification permission")
	}
}

func TestUpdateNotifyReminder_EnableDoesNotRequestPermission(t *testing.T) {
	app := newTestApp(t)
	provider := &notificationCapabilityProviderStub{
		capability: ports.NotificationCapability{
			PermissionState: ports.NotificationPermissionDenied,
			CanRequest:      false,
			CanOpenSettings: true,
		},
	}
	app.notificationCapability = provider

	notify := "notify"
	disabled := false
	created, err := app.CreateReminder(ReminderCreateInput{
		Name:         "Hydrate",
		IntervalSec:  1500,
		BreakSec:     1,
		Enabled:      &disabled,
		ReminderType: &notify,
	})
	if err != nil {
		t.Fatalf("CreateReminder() err=%v", err)
	}
	if len(created) != 1 {
		t.Fatalf("expected one reminder, got=%d", len(created))
	}

	enabled := true
	updated, err := app.UpdateReminder(ReminderPatch{ID: created[0].ID, Enabled: &enabled})
	if err != nil {
		t.Fatalf("UpdateReminder() err=%v", err)
	}
	if len(updated) != 1 || !updated[0].Enabled {
		t.Fatalf("unexpected update result: %+v", updated)
	}
	if provider.requested {
		t.Fatalf("update reminder should not request notification permission")
	}
}
