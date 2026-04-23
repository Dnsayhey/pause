package app

import (
	"context"
	"sync/atomic"

	"pause/internal/backend/bootstrap"
	analyticsdomain "pause/internal/backend/domain/analytics"
	reminderdomain "pause/internal/backend/domain/reminder"
	"pause/internal/backend/domain/settings"
	"pause/internal/backend/ports"
)

type App struct {
	ctx                    context.Context
	engine                 engineRuntime
	runtime                runtimeCloser
	reminders              reminderService
	analytics              analyticsService
	settingsSvc            settingsService
	notifier               ports.Notifier
	notificationCapability ports.NotificationCapabilityProvider
	desktop                desktopController
	quitRequested          atomic.Bool
}

type engineRuntime = bootstrap.RuntimeEngine
type skipMode = bootstrap.SkipMode

const (
	skipModeNormal    skipMode = bootstrap.SkipModeNormal
	skipModeEmergency skipMode = bootstrap.SkipModeEmergency
)

type runtimeCloser interface {
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
