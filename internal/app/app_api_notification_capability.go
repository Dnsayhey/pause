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
	logx.Infof("notification.permission_request started")
	if a == nil || a.notificationCapability == nil {
		capability := notificationCapabilityFromPorts(defaultNotificationCapability())
		logx.Infof(
			"notification.permission_request completed permission_state=%s can_request=%t can_open_settings=%t reason=%q provider=default",
			capability.PermissionState,
			capability.CanRequest,
			capability.CanOpenSettings,
			capability.Reason,
		)
		return capability, nil
	}
	capability, err := a.notificationCapability.RequestNotificationPermission()
	if err != nil {
		logx.Warnf("notification.permission_request failed err=%v", err)
		return NotificationCapability{}, err
	}
	result := notificationCapabilityFromPorts(normalizeNotificationCapability(capability))
	logx.Infof(
		"notification.permission_request completed permission_state=%s can_request=%t can_open_settings=%t reason=%q",
		result.PermissionState,
		result.CanRequest,
		result.CanOpenSettings,
		result.Reason,
	)
	return result, nil
}

func (a *App) OpenNotificationSettings() error {
	logx.Infof("notification.settings_open started")
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
	logx.Infof("notification.settings_open completed")
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
