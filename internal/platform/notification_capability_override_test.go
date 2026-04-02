package platform

import (
	"errors"
	"testing"

	"pause/internal/backend/ports"
	"pause/internal/platform/api"
)

func TestDisabledNotificationCapabilityProvider(t *testing.T) {
	provider := disabledNotificationCapabilityProvider{reason: "dev disabled"}
	got := provider.GetNotificationCapability()
	if got.PermissionState != ports.NotificationPermissionUnknown {
		t.Fatalf("permissionState=%q want=%q", got.PermissionState, ports.NotificationPermissionUnknown)
	}
	if got.CanRequest {
		t.Fatalf("CanRequest should be false")
	}
	if got.CanOpenSettings {
		t.Fatalf("CanOpenSettings should be false")
	}
	if got.Reason != "dev disabled" {
		t.Fatalf("Reason=%q want=%q", got.Reason, "dev disabled")
	}
}

func TestDisabledNotificationCapabilityProvider_OpenSettingsUnavailable(t *testing.T) {
	provider := disabledNotificationCapabilityProvider{reason: "dev disabled"}
	if err := provider.OpenNotificationSettings(); !errors.Is(err, api.ErrNotificationSettingsUnavailable) {
		t.Fatalf("OpenNotificationSettings() err=%v want=%v", err, api.ErrNotificationSettingsUnavailable)
	}
}
