package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	engineadapter "pause/internal/backend/adapters/engine"
	"pause/internal/backend/bootstrap"
	analyticsdomain "pause/internal/backend/domain/analytics"
	reminderdomain "pause/internal/backend/domain/reminder"
	settingsusecase "pause/internal/backend/usecase/settings"
	"pause/internal/core/history"
	"pause/internal/core/service"
	"pause/internal/core/settings"
	"pause/internal/core/state"
	"pause/internal/logx"
	"pause/internal/meta"
	"pause/internal/paths"
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

func NewApp(configPath string) (*App, error) {
	if configPath == "" {
		resolved, err := defaultConfigPath()
		if err != nil {
			return nil, err
		}
		configPath = resolved
	}

	store, err := settings.OpenSettingsStore(configPath)
	if err != nil {
		return nil, err
	}
	historyPath := defaultHistoryPath(configPath)
	historyStore, err := history.OpenHistoryStore(context.Background(), historyPath)
	if err != nil {
		return nil, err
	}
	if store.WasCreated() {
		language := resolveEffectiveLanguage(store.Get().UI.Language)
		if err := ensureBuiltInRemindersForFirstInstall(context.Background(), historyStore, language); err != nil {
			_ = historyStore.Close()
			return nil, err
		}
	}
	container, err := bootstrap.NewContainer(historyStore)
	if err != nil {
		_ = historyStore.Close()
		return nil, err
	}

	adapters := platform.NewAdapters(meta.EffectiveAppBundleID())
	engine := service.NewEngine(
		store,
		adapters.IdleProvider,
		adapters.LockStateProvider,
		adapters.SoundPlayer,
		adapters.StartupManager,
		historyStore,
	)
	engine.SetNotifier(adapters.Notifier)
	defs, err := container.ReminderService.List(context.Background())
	if err != nil {
		_ = historyStore.Close()
		return nil, err
	}
	engineReminders := reminderDefsToConfig(defs)
	engine.SetReminderConfigs(engineReminders)
	logx.Infof("app.reminders_synced source=usecase count=%d", len(engineReminders))
	settingsRepo := engineadapter.NewSettingsRepository(engine)
	settingsSvc, err := settingsusecase.NewService(settingsRepo)
	if err != nil {
		_ = historyStore.Close()
		return nil, err
	}

	logx.Infof(
		"app.init bundle_id=%s config_path=%s history_path=%s config_created=%t",
		meta.EffectiveAppBundleID(),
		configPath,
		historyPath,
		store.WasCreated(),
	)

	return &App{
		engine:      engine,
		history:     historyStore,
		reminders:   container.ReminderService,
		analytics:   container.AnalyticsService,
		settingsSvc: settingsSvc,
		notifier:    adapters.Notifier,
		desktop:     newDesktopController(),
	}, nil
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	if a.settingsSvc != nil {
		if err := a.settingsSvc.SyncPlatformSettings(appContextOrBackground(ctx)); err != nil {
			logx.Warnf("app.startup sync_platform_settings_err=%v", err)
		}
	} else {
		logx.Warnf("app.startup sync_platform_settings_skipped reason=settings_service_unavailable")
	}
	a.engine.Start(ctx)
	if a.desktop != nil {
		a.desktop.OnStartup(ctx, a)
	}
	logx.Infof("app.startup completed")
}

func (a *App) GetRuntimeState() state.RuntimeState {
	runtimeState := a.engine.GetRuntimeState(time.Now())
	return a.decorateRuntimeState(runtimeState)
}

func (a *App) Pause() (state.RuntimeState, error) {
	runtimeState, err := a.engine.Pause(time.Now())
	if err != nil {
		return state.RuntimeState{}, err
	}
	return a.decorateRuntimeState(runtimeState), nil
}

func (a *App) Resume() state.RuntimeState {
	return a.decorateRuntimeState(a.engine.Resume(time.Now()))
}

func (a *App) PauseReminder(reminderID int64) (state.RuntimeState, error) {
	if reminderID <= 0 {
		return state.RuntimeState{}, errors.New("reminder id is required")
	}
	runtimeState, err := a.engine.PauseReminder(reminderID, time.Now())
	if err != nil {
		return state.RuntimeState{}, err
	}
	return a.decorateRuntimeState(runtimeState), nil
}

func (a *App) ResumeReminder(reminderID int64) (state.RuntimeState, error) {
	if reminderID <= 0 {
		return state.RuntimeState{}, errors.New("reminder id is required")
	}
	runtimeState, err := a.engine.ResumeReminder(reminderID, time.Now())
	if err != nil {
		return state.RuntimeState{}, err
	}
	return a.decorateRuntimeState(runtimeState), nil
}

func (a *App) SkipCurrentBreak() (state.RuntimeState, error) {
	return a.skipCurrentBreakWithMode(service.SkipModeNormal)
}

func (a *App) skipCurrentBreakEmergency() (state.RuntimeState, error) {
	return a.skipCurrentBreakWithMode(service.SkipModeEmergency)
}

func (a *App) skipCurrentBreakWithMode(mode service.SkipMode) (state.RuntimeState, error) {
	runtimeState, err := a.engine.SkipCurrentBreak(time.Now(), mode)
	if err != nil {
		return state.RuntimeState{}, err
	}
	return a.decorateRuntimeState(runtimeState), nil
}

func (a *App) StartBreakNow() (state.RuntimeState, error) {
	runtimeState, err := a.engine.StartBreakNow(time.Now())
	if err != nil {
		return state.RuntimeState{}, err
	}
	return a.decorateRuntimeState(runtimeState), nil
}

func (a *App) StartBreakNowForReason(reminderID int64) (state.RuntimeState, error) {
	if reminderID < 0 {
		return state.RuntimeState{}, errors.New("reminder id is invalid")
	}
	runtimeState, err := a.engine.StartBreakNowForReason(reminderID, time.Now())
	if err != nil {
		return state.RuntimeState{}, err
	}
	return a.decorateRuntimeState(runtimeState), nil
}

func (a *App) SendBreakFallbackNotification(state state.RuntimeState) {
	if a.notifier == nil {
		logx.Warnf("overlay.fallback_notification_skipped reason=no_notifier")
		return
	}
	reasons := "none"
	if state.CurrentSession != nil {
		reasons = joinReasons(state.CurrentSession.Reasons)
	}
	body := buildBreakNotificationBody(state)
	notifier := a.notifier
	go func(reasonKey string, n platform.Notifier, message string) {
		if err := n.ShowReminder("Time to rest", message); err != nil {
			logx.Warnf("overlay.fallback_notification_err err=%v", err)
			return
		}
		logx.Infof("overlay.fallback_notification_sent reasons=%s", reasonKey)
	}(reasons, notifier, body)
}

func buildBreakNotificationBody(state state.RuntimeState) string {
	if state.CurrentSession == nil {
		return "Break started"
	}

	namesByReason := map[int64]string{}
	for _, reminder := range state.Reminders {
		id := reminder.ID
		name := strings.TrimSpace(reminder.Name)
		if id <= 0 || name == "" {
			continue
		}
		namesByReason[id] = name
	}

	parts := make([]string, 0, len(state.CurrentSession.Reasons))
	for _, reason := range state.CurrentSession.Reasons {
		if reason <= 0 {
			continue
		}
		if name, ok := namesByReason[reason]; ok && name != "" {
			parts = append(parts, name)
		}
	}

	label := strings.Join(parts, " + ")

	if state.CurrentSession.RemainingSec > 0 {
		duration := (time.Duration(state.CurrentSession.RemainingSec) * time.Second).String()
		return fmt.Sprintf("%s break for %s", label, duration)
	}
	return label + " break started"
}

func joinReasons(reasons []int64) string {
	if len(reasons) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		parts = append(parts, fmt.Sprintf("%d", reason))
	}
	return strings.Join(parts, "+")
}

func defaultConfigPath() (string, error) {
	return paths.ConfigFile("settings.json")
}

func defaultHistoryPath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "history.db")
}

func appContextOrBackground(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

func (a *App) Shutdown(_ context.Context) {
	if a == nil || a.history == nil {
		return
	}
	if err := a.history.Close(); err != nil {
		logx.Warnf("app.shutdown history_close_err=%v", err)
	}
}

func (a *App) decorateRuntimeState(state state.RuntimeState) state.RuntimeState {
	settings := a.engine.GetSettings()
	state.EffectiveLanguage = resolveEffectiveLanguage(settings.UI.Language)
	state.EffectiveTheme = resolveEffectiveTheme(settings.UI.Theme)
	return decorateRuntimeStateForPlatform(state)
}
