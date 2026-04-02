package platform

import (
	"os"
	"strings"

	"pause/internal/platform/api"
	"pause/internal/platform/fallbacks"
)

func withNotificationCapabilityOverride(adapters api.Adapters) api.Adapters {
	if reason, disabled := notificationCapabilityDisabledByBuild(); disabled {
		adapters.NotificationCapabilityProvider = fallbacks.DisabledNotificationCapabilityProvider{Reason: reason}
		return adapters
	}
	if reason, disabled := notificationCapabilityDisabledByEnv(); disabled {
		adapters.NotificationCapabilityProvider = fallbacks.DisabledNotificationCapabilityProvider{Reason: reason}
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
