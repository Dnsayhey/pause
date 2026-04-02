package platform

import "testing"

func TestNotificationCapabilityDisabledByEnv(t *testing.T) {
	t.Setenv("PAUSE_DISABLE_NOTIFICATION_CAPABILITY", "")
	if reason, disabled := notificationCapabilityDisabledByEnv(); disabled || reason != "" {
		t.Fatalf("expected notification capability to stay enabled by default")
	}

	t.Setenv("PAUSE_DISABLE_NOTIFICATION_CAPABILITY", "1")
	if reason, disabled := notificationCapabilityDisabledByEnv(); !disabled || reason == "" {
		t.Fatalf("expected env flag to disable notification capability")
	}

	t.Setenv("PAUSE_DISABLE_NOTIFICATION_CAPABILITY", "false")
	if reason, disabled := notificationCapabilityDisabledByEnv(); disabled || reason != "" {
		t.Fatalf("expected false value to keep notification capability enabled")
	}
}
