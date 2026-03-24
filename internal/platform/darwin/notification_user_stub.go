//go:build darwin && !cgo

package darwin

import "errors"

func showDarwinUserNotification(_, _ string) error {
	return errors.New("darwin user notification requires cgo")
}
