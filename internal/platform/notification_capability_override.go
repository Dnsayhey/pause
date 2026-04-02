package platform

import (
	"os"
	"strings"

	"pause/internal/backend/ports"
	"pause/internal/platform/api"
)

type disabledNotificationCapabilityProvider struct {
	reason string
}

func disabledNotificationCapability(reason string) ports.NotificationCapability {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "notification capability disabled"
	}
	return ports.NotificationCapability{
		PermissionState: ports.NotificationPermissionUnknown,
		CanRequest:      false,
		CanOpenSettings: false,
		Reason:          reason,
	}
}

func withNotificationCapabilityOverride(adapters api.Adapters) api.Adapters {
	if reason, disabled := notificationCapabilityDisabledByBuild(); disabled {
		adapters.NotificationCapabilityProvider = disabledNotificationCapabilityProvider{reason: reason}
		return adapters
	}
	if reason, disabled := notificationCapabilityDisabledByEnv(); disabled {
		adapters.NotificationCapabilityProvider = disabledNotificationCapabilityProvider{reason: reason}
	}
	return adapters
}

func (p disabledNotificationCapabilityProvider) GetNotificationCapability() ports.NotificationCapability {
	return disabledNotificationCapability(p.reason)
}

func (p disabledNotificationCapabilityProvider) RequestNotificationPermission() (ports.NotificationCapability, error) {
	return disabledNotificationCapability(p.reason), nil
}

func (p disabledNotificationCapabilityProvider) OpenNotificationSettings() error {
	return api.ErrNotificationSettingsUnavailable
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
