package app

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"pause/internal/core/config"
	"pause/internal/core/service"
	"pause/internal/logx"
	"pause/internal/meta"
	"pause/internal/paths"
	"pause/internal/platform"
)

type App struct {
	ctx           context.Context
	engine        *service.Engine
	notifier      platform.Notifier
	desktop       desktopController
	quitRequested atomic.Bool
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

	store, err := config.NewStore(configPath)
	if err != nil {
		return nil, err
	}

	adapters := platform.NewAdapters(meta.EffectiveAppBundleID())
	engine := service.NewEngine(
		store,
		adapters.IdleProvider,
		adapters.SoundPlayer,
		adapters.StartupManager,
	)

	logx.Infof(
		"app.init bundle_id=%s config_path=%s config_created=%t",
		meta.EffectiveAppBundleID(),
		configPath,
		store.WasCreated(),
	)

	return &App{
		engine:   engine,
		notifier: adapters.Notifier,
		desktop:  newDesktopController(),
	}, nil
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	if err := a.engine.SyncPlatformSettings(); err != nil {
		logx.Warnf("app.startup sync_platform_settings_err=%v", err)
	}
	a.engine.Start(ctx)
	if a.desktop != nil {
		a.desktop.OnStartup(ctx, a)
	}
	logx.Infof("app.startup completed")
}

func (a *App) GetSettings() config.Settings {
	return a.engine.GetSettings()
}

func (a *App) UpdateSettings(patch config.SettingsPatch) (config.Settings, error) {
	settings, err := a.engine.UpdateSettings(patch)
	if err != nil {
		logx.Warnf("app.update_settings_err err=%v", err)
		return config.Settings{}, err
	}
	return settings, nil
}

func (a *App) GetRuntimeState() config.RuntimeState {
	state := a.engine.GetRuntimeState(time.Now())
	return a.decorateRuntimeState(state)
}

func (a *App) Pause(mode string, durationSec int) (config.RuntimeState, error) {
	if mode == service.PauseModeTemporary && durationSec == 0 {
		durationSec = 15 * 60
	}
	state, err := a.engine.Pause(mode, durationSec, time.Now())
	if err != nil {
		return config.RuntimeState{}, err
	}
	return a.decorateRuntimeState(state), nil
}

func (a *App) Resume() config.RuntimeState {
	return a.decorateRuntimeState(a.engine.Resume(time.Now()))
}

func (a *App) SkipCurrentBreak() (config.RuntimeState, error) {
	return a.skipCurrentBreakWithMode(service.SkipModeNormal)
}

func (a *App) skipCurrentBreakEmergency() (config.RuntimeState, error) {
	return a.skipCurrentBreakWithMode(service.SkipModeEmergency)
}

func (a *App) skipCurrentBreakWithMode(mode service.SkipMode) (config.RuntimeState, error) {
	state, err := a.engine.SkipCurrentBreak(time.Now(), mode)
	if err != nil {
		return config.RuntimeState{}, err
	}
	return a.decorateRuntimeState(state), nil
}

func (a *App) StartBreakNow() (config.RuntimeState, error) {
	state, err := a.engine.StartBreakNow(time.Now())
	if err != nil {
		return config.RuntimeState{}, err
	}
	return a.decorateRuntimeState(state), nil
}

func (a *App) StartBreakNowForReason(reason string) (config.RuntimeState, error) {
	state, err := a.engine.StartBreakNowForReason(reason, time.Now())
	if err != nil {
		return config.RuntimeState{}, err
	}
	return a.decorateRuntimeState(state), nil
}

func (a *App) GetLaunchAtLogin() (bool, error) {
	return a.engine.GetLaunchAtLogin()
}

func (a *App) SetLaunchAtLogin(enabled bool) (bool, error) {
	return a.engine.SetLaunchAtLogin(enabled)
}

func (a *App) SendBreakFallbackNotification(state config.RuntimeState) {
	if a.notifier == nil {
		logx.Warnf("overlay.fallback_notification_skipped reason=no_notifier")
		return
	}
	if err := a.notifier.ShowReminder("Time to rest", buildBreakNotificationBody(state)); err != nil {
		logx.Warnf("overlay.fallback_notification_err err=%v", err)
		return
	}
	reasons := "none"
	if state.CurrentSession != nil {
		reasons = joinReasons(state.CurrentSession.Reasons)
	}
	logx.Warnf("overlay.fallback_notification_sent reasons=%s", reasons)
}

func buildBreakNotificationBody(state config.RuntimeState) string {
	if state.CurrentSession == nil {
		return "Break started"
	}

	parts := make([]string, 0, len(state.CurrentSession.Reasons))
	for _, reason := range state.CurrentSession.Reasons {
		switch strings.ToLower(strings.TrimSpace(reason)) {
		case "eye":
			parts = append(parts, "Eye")
		case "stand":
			parts = append(parts, "Stand")
		}
	}

	label := "Break"
	if len(parts) > 0 {
		label = strings.Join(parts, " + ")
	}

	if state.CurrentSession.RemainingSec > 0 {
		return fmt.Sprintf("%s break for %s", label, (time.Duration(state.CurrentSession.RemainingSec) * time.Second).String())
	}
	return label + " break started"
}

func joinReasons(reasons []string) string {
	if len(reasons) == 0 {
		return "none"
	}
	return strings.Join(reasons, "+")
}

func defaultConfigPath() (string, error) {
	return paths.ConfigFile("settings.json")
}

func (a *App) decorateRuntimeState(state config.RuntimeState) config.RuntimeState {
	settings := a.engine.GetSettings()
	state.EffectiveLanguage = resolveEffectiveLanguage(settings.UI.Language)
	state.EffectiveTheme = resolveEffectiveTheme(settings.UI.Theme)
	return decorateRuntimeStateForPlatform(state)
}
