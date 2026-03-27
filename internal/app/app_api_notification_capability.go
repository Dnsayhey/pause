package app

import (
	"errors"
	"strings"

	"pause/internal/backend/ports"
	"pause/internal/logx"
)

func (a *App) GetNotificationCapability() NotificationCapability {
	capability := a.notificationCapabilityFromProvider()
	return notificationCapabilityFromPorts(capability)
}

func (a *App) RequestNotificationPermission() (NotificationCapability, error) {
	if a == nil || a.notificationCapability == nil {
		return notificationCapabilityFromPorts(defaultNotificationCapability()), nil
	}
	capability, err := a.notificationCapability.RequestNotificationPermission()
	if err != nil {
		logx.Warnf("notification.permission_request failed err=%v", err)
		return NotificationCapability{}, err
	}
	return notificationCapabilityFromPorts(normalizeNotificationCapability(capability)), nil
}

func (a *App) OpenNotificationSettings() error {
	if a == nil || a.notificationCapability == nil {
		err := errors.New("notification settings are unavailable on this platform")
		logx.Warnf("notification.settings_open failed err=%v", err)
		return err
	}
	err := a.notificationCapability.OpenNotificationSettings()
	if err != nil {
		logx.Warnf("notification.settings_open failed err=%v", err)
		return err
	}
	return nil
}

func (a *App) notificationCapabilityFromProvider() ports.NotificationCapability {
	if a == nil || a.notificationCapability == nil {
		return defaultNotificationCapability()
	}
	return normalizeNotificationCapability(a.notificationCapability.GetNotificationCapability())
}

func defaultNotificationCapability() ports.NotificationCapability {
	return ports.NotificationCapability{
		PermissionState: ports.NotificationPermissionUnknown,
		CanRequest:      false,
		CanOpenSettings: false,
		Reason:          "notification capability unavailable",
	}
}

func normalizeNotificationCapability(capability ports.NotificationCapability) ports.NotificationCapability {
	result := capability
	switch result.PermissionState {
	case ports.NotificationPermissionAuthorized,
		ports.NotificationPermissionNotDetermined,
		ports.NotificationPermissionDenied,
		ports.NotificationPermissionRestricted,
		ports.NotificationPermissionUnknown:
	default:
		result.PermissionState = ports.NotificationPermissionUnknown
	}
	result.Reason = strings.TrimSpace(result.Reason)
	return result
}
