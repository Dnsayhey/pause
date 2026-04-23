package ports

import (
	"context"
	"time"

	"pause/internal/backend/domain/settings"
)

type BreakRecordInput struct {
	StartedAt       time.Time
	EndedAt         time.Time
	Source          string
	PlannedBreakSec int
	ActualBreakSec  int
	Skipped         bool
	ReminderIDs     []int64
}

type BreakRecorder interface {
	RecordBreak(ctx context.Context, input BreakRecordInput) error
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
