package engine

import (
	"context"
	"sync"
	"time"

	"pause/internal/backend/domain/reminder"
	"pause/internal/backend/domain/settings"
	"pause/internal/backend/ports"
	"pause/internal/backend/runtime/scheduler"
	"pause/internal/backend/runtime/session"
)

type SkipMode string

const (
	SkipModeNormal    SkipMode = "normal"
	SkipModeEmergency SkipMode = "emergency"
)

type SettingsStore interface {
	Get() settings.Settings
	Update(patch settings.SettingsPatch) (settings.Settings, error)
}

type noopIdleProvider struct{}

func (noopIdleProvider) CurrentIdleSeconds() int { return 0 }

type noopLockStateProvider struct{}

func (noopLockStateProvider) IsScreenLocked() bool { return false }

type noopSoundPlayer struct{}

func (noopSoundPlayer) PlayBreakEnd(settings.SoundSettings) error { return nil }

type noopNotifier struct{}

func (noopNotifier) ShowReminder(context.Context, string, string) error { return nil }

type pendingHistoryBreak struct {
	StartedAt       time.Time
	Source          string
	PlannedBreakSec int
	ReminderIDs     []int64
}

type Engine struct {
	mu        sync.Mutex
	startOnce sync.Once
	stopOnce  sync.Once

	store     SettingsStore
	reminders []reminder.Reminder
	scheduler *scheduler.Scheduler
	session   *session.Manager
	history   ports.BreakRepository

	idleProvider ports.IdleProvider
	lockProvider ports.LockStateProvider
	soundPlayer  ports.SoundPlayer
	notifier     ports.Notifier

	lastTick      time.Time
	tickRemainder time.Duration

	pausedReminder map[int64]bool
	globalEnabled  bool

	lastTickActive bool
	currentIdleSec int
	currentLocked  bool

	activeHistoryBreak *pendingHistoryBreak
	runCtx             context.Context
	cancelRun          context.CancelFunc
	backgroundTasks    sync.WaitGroup
}

func NewEngine(
	store SettingsStore,
	idleProvider ports.IdleProvider,
	lockProvider ports.LockStateProvider,
	soundPlayer ports.SoundPlayer,
	notifier ports.Notifier,
	history ports.BreakRepository,
) *Engine {
	if idleProvider == nil {
		idleProvider = noopIdleProvider{}
	}
	if lockProvider == nil {
		lockProvider = noopLockStateProvider{}
	}
	if soundPlayer == nil {
		soundPlayer = noopSoundPlayer{}
	}
	if notifier == nil {
		notifier = noopNotifier{}
	}
	return &Engine{
		store:          store,
		reminders:      cloneReminderConfigs(nil),
		scheduler:      scheduler.New(),
		session:        session.NewManager(),
		history:        history,
		idleProvider:   idleProvider,
		lockProvider:   lockProvider,
		soundPlayer:    soundPlayer,
		notifier:       notifier,
		pausedReminder: map[int64]bool{},
		globalEnabled:  true,
	}
}
