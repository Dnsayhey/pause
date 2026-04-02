package api

import (
	"errors"
	"testing"

	"pause/internal/backend/ports"
)

func TestUnsupportedNotificationCapability(t *testing.T) {
	got := UnsupportedNotificationCapability("")
	if got.PermissionState != ports.NotificationPermissionUnknown {
		t.Fatalf("permissionState=%q want=%q", got.PermissionState, ports.NotificationPermissionUnknown)
	}
	if got.CanRequest {
		t.Fatalf("CanRequest should be false")
	}
	if got.CanOpenSettings {
		t.Fatalf("CanOpenSettings should be false")
	}
	if got.Reason == "" {
		t.Fatalf("Reason should not be empty")
	}
}

func TestNoopNotificationCapabilityProvider_OpenSettingsUnavailable(t *testing.T) {
	provider := NoopNotificationCapabilityProvider{}
	if err := provider.OpenNotificationSettings(); !errors.Is(err, ErrNotificationSettingsUnavailable) {
		t.Fatalf("OpenNotificationSettings() err=%v want=%v", err, ErrNotificationSettingsUnavailable)
	}
}
