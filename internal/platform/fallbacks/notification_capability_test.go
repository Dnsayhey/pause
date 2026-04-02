package fallbacks

import (
	"testing"

	"pause/internal/backend/ports"
)

func TestDisabledNotificationCapability(t *testing.T) {
	got := DisabledNotificationCapability("dev disabled")
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
