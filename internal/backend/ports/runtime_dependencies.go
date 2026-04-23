package ports

import (
	"context"
	"time"

	"pause/internal/backend/domain/settings"
)

type BreakRepository interface {
	RecordBreak(
		ctx context.Context,
		startedAt time.Time,
		endedAt time.Time,
		source string,
		plannedBreakSec int,
		actualBreakSec int,
		skipped bool,
		reminderIDs []int64,
	) error
}

type IdleProvider interface {
	CurrentIdleSeconds() int
}

type LockStateProvider interface {
	IsScreenLocked() bool
}

type Notifier interface {
	ShowReminder(ctx context.Context, title, body string) error
}

type NotificationPermissionState string

const (
	NotificationPermissionAuthorized    NotificationPermissionState = "authorized"
	NotificationPermissionNotDetermined NotificationPermissionState = "not_determined"
	NotificationPermissionDenied        NotificationPermissionState = "denied"
	NotificationPermissionRestricted    NotificationPermissionState = "restricted"
	NotificationPermissionUnknown       NotificationPermissionState = "unknown"
)

type NotificationCapability struct {
	PermissionState NotificationPermissionState
	CanRequest      bool
	CanOpenSettings bool
	Reason          string
}

type NotificationCapabilityProvider interface {
	GetNotificationCapability() NotificationCapability
	RequestNotificationPermission() (NotificationCapability, error)
	OpenNotificationSettings() error
}

type SoundPlayer interface {
	PlayBreakEnd(sound settings.SoundSettings) error
}

type StartupManager interface {
	SetLaunchAtLogin(enabled bool) error
	GetLaunchAtLogin() (bool, error)
}

type Clock interface {
	Now() time.Time
}
