package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"pause/internal/config"
	"pause/internal/diag"
	"pause/internal/platform"
	"pause/internal/service"
)

type App struct {
	ctx     context.Context
	engine  *service.Engine
	desktop desktopController
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
		adapters.Notifier,
		adapters.SoundPlayer,
		adapters.StartupManager,
	)

	return &App{
		engine:  engine,
		desktop: newDesktopController(),
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
