package app

import (
	"context"
	"sync/atomic"

	analyticsdomain "pause/internal/backend/domain/analytics"
	reminderdomain "pause/internal/backend/domain/reminder"
	"pause/internal/core/history"
	"pause/internal/core/service"
	"pause/internal/core/settings"
	"pause/internal/platform"
)

type App struct {
	ctx           context.Context
	engine        *service.Engine
	history       *history.HistoryStore
	reminders     reminderService
	analytics     analyticsService
	settingsSvc   settingsService
	notifier      platform.Notifier
	desktop       desktopController
	quitRequested atomic.Bool
}

type reminderService interface {
	List(ctx context.Context) ([]reminderdomain.Reminder, error)
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
