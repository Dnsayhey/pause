package app

import (
	"context"
	"sync/atomic"
	"time"

	analyticsdomain "pause/internal/backend/domain/analytics"
	reminderdomain "pause/internal/backend/domain/reminder"
	"pause/internal/backend/runtime/state"
	corereminder "pause/internal/core/reminder"
	"pause/internal/core/settings"
	"pause/internal/platform"
)

type App struct {
	ctx           context.Context
	engine        engineRuntime
	history       historyCloser
	reminders     reminderService
	analytics     analyticsService
	settingsSvc   settingsService
	notifier      platform.Notifier
	desktop       desktopController
	quitRequested atomic.Bool
}

type engineRuntime interface {
	Start(ctx context.Context)
	GetSettings() settings.Settings
	GetRuntimeState(now time.Time) state.RuntimeState
	Pause(now time.Time) (state.RuntimeState, error)
	Resume(now time.Time) state.RuntimeState
	PauseReminder(reminderID int64, now time.Time) (state.RuntimeState, error)
	ResumeReminder(reminderID int64, now time.Time) (state.RuntimeState, error)
	SkipCurrentBreak(now time.Time, mode skipMode) (state.RuntimeState, error)
	StartBreakNow(now time.Time) (state.RuntimeState, error)
	StartBreakNowForReason(reason int64, now time.Time) (state.RuntimeState, error)
	SetReminderConfigs(reminders []corereminder.ReminderConfig) []corereminder.ReminderConfig
}

type skipMode string

const (
	skipModeNormal    skipMode = "normal"
	skipModeEmergency skipMode = "emergency"
)

type historyCloser interface {
	Close() error
}

type reminderService interface {
	List(ctx context.Context) ([]reminderdomain.Reminder, error)
	EnsureDefaults(ctx context.Context, inputs []reminderdomain.CreateInput) error
	Create(ctx context.Context, input reminderdomain.CreateInput) ([]reminderdomain.Reminder, error)
	Update(ctx context.Context, patch reminderdomain.Patch) ([]reminderdomain.Reminder, error)
	Delete(ctx context.Context, reminderID int64) ([]reminderdomain.Reminder, error)
}

type analyticsService interface {
	GetWeeklyStats(ctx context.Context, fromSec int64, toSec int64) (analyticsdomain.WeeklyStats, error)
	GetSummary(ctx context.Context, fromSec int64, toSec int64) (analyticsdomain.Summary, error)
	GetTrendByDay(ctx context.Context, fromSec int64, toSec int64) (analyticsdomain.Trend, error)
	GetBreakTypeDistribution(ctx context.Context, fromSec int64, toSec int64) (analyticsdomain.BreakTypeDistribution, error)
}

type settingsService interface {
	Get(ctx context.Context) settings.Settings
	Update(ctx context.Context, patch settings.SettingsPatch) (settings.Settings, error)
	SyncPlatformSettings(ctx context.Context) error
	GetLaunchAtLogin(ctx context.Context) (bool, error)
	SetLaunchAtLogin(ctx context.Context, enabled bool) (bool, error)
}

type desktopController interface {
	OnStartup(ctx context.Context, app *App)
}
