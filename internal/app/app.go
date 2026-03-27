package app

import (
	"context"

	"pause/internal/backend/bootstrap"
	"pause/internal/logx"
	"pause/internal/meta"
	"pause/internal/paths"
)

func NewApp(configPath string) (*App, error) {
	if configPath == "" {
		resolved, err := defaultConfigPath()
		if err != nil {
			return nil, err
		}
		configPath = resolved
	}

	runtime, err := bootstrap.NewRuntime(configPath, meta.EffectiveAppBundleID())
	if err != nil {
		return nil, err
	}
	if runtime.Settings.WasCreated() {
		language := resolveEffectiveLanguage(runtime.Settings.Get().UI.Language)
		if err := ensureBuiltInRemindersForFirstInstall(context.Background(), runtime.ReminderService, language); err != nil {
			_ = runtime.Close()
			return nil, err
		}
	}

	logx.Infof(
		"app.init bundle_id=%s config_path=%s history_path=%s config_created=%t",
		meta.EffectiveAppBundleID(),
		configPath,
		runtime.HistoryPath,
		runtime.Settings.WasCreated(),
	)

	return &App{
		engine:                 runtime.Engine,
		history:                runtime.History,
		reminders:              runtime.ReminderService,
		analytics:              runtime.AnalyticsService,
		settingsSvc:            runtime.SettingsService,
		notifier:               runtime.Notifier,
		notificationCapability: runtime.NotificationCapabilityProvider,
		desktop:                newDesktopController(),
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
	logx.Infof("app.shutdown started")
	if err := a.history.Close(); err != nil {
		logx.Warnf("app.shutdown history_close_err=%v", err)
		return
	}
	logx.Infof("app.shutdown completed")
}
