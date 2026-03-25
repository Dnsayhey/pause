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
	ShowReminder(title, body string) error
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
