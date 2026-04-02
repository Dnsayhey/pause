//go:build !dev

package platform

import "testing"

func TestNotificationCapabilityDisabledByBuildDefault(t *testing.T) {
	if reason, disabled := notificationCapabilityDisabledByBuild(); disabled || reason != "" {
		t.Fatalf("expected notification capability to stay enabled outside dev builds")
	}
}
