//go:build darwin && !cgo

package darwin

import "errors"

const (
	darwinNotificationStatusNotDetermined = 0
	darwinNotificationStatusDenied        = 1
	darwinNotificationStatusAuthorized    = 2
	darwinNotificationStatusProvisional   = 3
	darwinNotificationStatusEphemeral     = 4
)

func showDarwinUserNotification(_, _ string) error {
	return errors.New("darwin user notification requires cgo")
}

func darwinNotificationAuthorizationStatus() (int, error) {
	return darwinNotificationStatusUnknown(), errors.New("darwin notification status requires cgo")
}

func darwinRequestNotificationAuthorization() (bool, error) {
	return false, errors.New("darwin notification authorization requires cgo")
}

func darwinOpenNotificationSettings(_ string) error {
	return errors.New("darwin notification settings requires cgo")
}

func installDarwinNotificationClickDelegate() error {
	return errors.New("darwin notification delegate requires cgo")
}

func darwinNotificationStatusUnknown() int {
	return -1
}
