package app

import (
	"errors"
	"testing"

	"pause/internal/backend/ports"
)

func TestGetNotificationCapability_DefaultWhenProviderMissing(t *testing.T) {
	a := &App{}
	got := a.GetNotificationCapability()
	if got.PermissionState != string(ports.NotificationPermissionUnknown) {
		t.Fatalf("permissionState=%q want=%q", got.PermissionState, ports.NotificationPermissionUnknown)
	}
	if got.CanRequest {
		t.Fatalf("canRequest should be false")
	}
}

func TestGetNotificationCapability_FromProvider(t *testing.T) {
	a := &App{
		notificationCapability: &notificationCapabilityProviderStub{
			capability: ports.NotificationCapability{
				PermissionState: ports.NotificationPermissionDenied,
				CanRequest:      false,
				CanOpenSettings: true,
				Reason:          "denied by user",
			},
		},
	}
	got := a.GetNotificationCapability()
	if got.PermissionState != string(ports.NotificationPermissionDenied) {
		t.Fatalf("permissionState=%q want=%q", got.PermissionState, ports.NotificationPermissionDenied)
	}
	if !got.CanOpenSettings {
		t.Fatalf("canOpenSettings should be true")
	}
	if got.Reason != "denied by user" {
		t.Fatalf("reason=%q want=%q", got.Reason, "denied by user")
	}
}

func TestRequestNotificationPermission_DefaultWhenProviderMissing(t *testing.T) {
	a := &App{}
	got, err := a.RequestNotificationPermission()
	if err != nil {
		t.Fatalf("RequestNotificationPermission() err=%v", err)
	}
	if got.PermissionState != string(ports.NotificationPermissionUnknown) {
		t.Fatalf("permissionState=%q want=%q", got.PermissionState, ports.NotificationPermissionUnknown)
	}
}

func TestRequestNotificationPermission_FromProvider(t *testing.T) {
	stub := &notificationCapabilityProviderStub{
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
	a := &App{notificationCapability: stub}
	got, err := a.RequestNotificationPermission()
	if err != nil {
		t.Fatalf("RequestNotificationPermission() err=%v", err)
	}
	if !stub.requested {
		t.Fatalf("expected notification permission request")
	}
	if got.PermissionState != string(ports.NotificationPermissionAuthorized) {
		t.Fatalf("permissionState=%q want=%q", got.PermissionState, ports.NotificationPermissionAuthorized)
	}
}

func TestRequestNotificationPermission_PropagatesError(t *testing.T) {
	wantErr := errors.New("request failed")
	stub := &notificationCapabilityProviderStub{requestErr: wantErr}
	a := &App{notificationCapability: stub}
	if _, err := a.RequestNotificationPermission(); !errors.Is(err, wantErr) {
		t.Fatalf("RequestNotificationPermission() err=%v want=%v", err, wantErr)
	}
}

func TestOpenNotificationSettings(t *testing.T) {
	stub := &notificationCapabilityProviderStub{}
	a := &App{notificationCapability: stub}
	if err := a.OpenNotificationSettings(); err != nil {
		t.Fatalf("OpenNotificationSettings() err=%v", err)
	}

	wantErr := errors.New("open settings failed")
	stub.openSettingsErr = wantErr
	if err := a.OpenNotificationSettings(); !errors.Is(err, wantErr) {
		t.Fatalf("OpenNotificationSettings() err=%v want=%v", err, wantErr)
	}
}
