package api

import (
	"strings"

	"pause/internal/backend/ports"
)

type DisabledNotificationCapabilityProvider struct {
	Reason string
}

func DisabledNotificationCapability(reason string) ports.NotificationCapability {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "notification capability unavailable"
	}
	return ports.NotificationCapability{
		PermissionState: ports.NotificationPermissionUnknown,
		CanRequest:      false,
		CanOpenSettings: false,
		Reason:          reason,
	}
}

func (p DisabledNotificationCapabilityProvider) GetNotificationCapability() ports.NotificationCapability {
	return DisabledNotificationCapability(p.Reason)
}

func (p DisabledNotificationCapabilityProvider) RequestNotificationPermission() (ports.NotificationCapability, error) {
	return DisabledNotificationCapability(p.Reason), nil
}

func (p DisabledNotificationCapabilityProvider) OpenNotificationSettings() error {
	return nil
}
