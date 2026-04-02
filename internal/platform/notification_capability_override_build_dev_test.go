//go:build dev

package platform

import "testing"

func TestNotificationCapabilityDisabledByBuildDev(t *testing.T) {
	if reason, disabled := notificationCapabilityDisabledByBuild(); !disabled || reason == "" {
		t.Fatalf("expected dev build to disable notification capability by default")
	}
}
