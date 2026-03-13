package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"pause/internal/config"
	"pause/internal/diag"
	"pause/internal/platform"
	"pause/internal/service"
)

type App struct {
	ctx      context.Context
	engine   *service.Engine
	notifier platform.Notifier
	desktop  desktopController
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

	adapters := platform.NewAdapters("com.pause.app")
	engine := service.NewEngine(
		store,
		adapters.IdleProvider,
		adapters.SoundPlayer,
		adapters.StartupManager,
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
		diag.Logf("app.startup sync_platform_settings_err=%v", err)
	}
	a.engine.Start(ctx)
	if a.desktop != nil {
		a.desktop.OnStartup(ctx, a)
	}
}

func (a *App) GetSettings() config.Settings {
	return a.engine.GetSettings()
}

func (a *App) UpdateSettings(patch config.SettingsPatch) (config.Settings, error) {
	return a.engine.UpdateSettings(patch)
}

func (a *App) GetRuntimeState() config.RuntimeState {
	state := a.engine.GetRuntimeState(time.Now())
	return decorateRuntimeStateForPlatform(state)
}

func (a *App) Pause(mode string, durationSec int) (config.RuntimeState, error) {
	if mode == service.PauseModeTemporary && durationSec == 0 {
		durationSec = 15 * 60
	}
	return a.engine.Pause(mode, durationSec, time.Now())
}

func (a *App) Resume() config.RuntimeState {
	return a.engine.Resume(time.Now())
}

func (a *App) SkipCurrentBreak() (config.RuntimeState, error) {
	return a.engine.SkipCurrentBreak(time.Now())
}

func (a *App) StartBreakNow() (config.RuntimeState, error) {
	return a.engine.StartBreakNow(time.Now())
}

func (a *App) StartBreakNowForReason(reason string) (config.RuntimeState, error) {
	return a.engine.StartBreakNowForReason(reason, time.Now())
}

func (a *App) GetLaunchAtLogin() (bool, error) {
	return a.engine.GetLaunchAtLogin()
}

func (a *App) SetLaunchAtLogin(enabled bool) (bool, error) {
	return a.engine.SetLaunchAtLogin(enabled)
}

func (a *App) SendBreakFallbackNotification(state config.RuntimeState) {
	if a.notifier == nil {
		return
	}
	_ = a.notifier.ShowReminder("Time to rest", buildBreakNotificationBody(state))
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

func defaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if home == "" {
		return "", errors.New("unable to resolve home directory")
	}

	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData != "" {
			return filepath.Join(appData, "Pause", "settings.json"), nil
		}
	}

	return filepath.Join(home, ".pause", "settings.json"), nil
}
