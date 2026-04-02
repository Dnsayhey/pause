package platform

import (
	"os"
	"strings"

	"pause/internal/platform/api"
)

func withNotificationCapabilityOverride(adapters api.Adapters) api.Adapters {
	if reason, disabled := notificationCapabilityDisabledByEnv(); disabled {
		adapters.NotificationCapabilityProvider = api.DisabledNotificationCapabilityProvider{Reason: reason}
	}
	return adapters
}

func notificationCapabilityDisabledByEnv() (string, bool) {
	raw := strings.TrimSpace(os.Getenv("PAUSE_DISABLE_NOTIFICATION_CAPABILITY"))
	if raw == "" {
		return "", false
	}
	switch strings.ToLower(raw) {
	case "0", "false", "no", "off":
		return "", false
	default:
		return "notification capability disabled by env: PAUSE_DISABLE_NOTIFICATION_CAPABILITY", true
	}
}
