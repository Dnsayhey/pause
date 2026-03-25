package app

import (
	"context"
	"path/filepath"

	"pause/internal/backend/bootstrap"
	"pause/internal/core/history"
	"pause/internal/core/service"
	"pause/internal/core/settings"
	"pause/internal/logx"
	"pause/internal/meta"
	"pause/internal/paths"
	"pause/internal/platform"
)

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
	container, err := bootstrap.NewContainer(historyStore)
	if err != nil {
		_ = historyStore.Close()
		return nil, err
	}
	if store.WasCreated() {
		language := resolveEffectiveLanguage(store.Get().UI.Language)
		if err := ensureBuiltInRemindersForFirstInstall(context.Background(), container.ReminderService, language); err != nil {
			_ = historyStore.Close()
			return nil, err
		}
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
	settingsSvc, err := bootstrap.NewSettingsService(engine)
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
