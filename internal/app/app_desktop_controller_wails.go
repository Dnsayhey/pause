//go:build wails

package app

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"pause/internal/desktop"
	"pause/internal/logx"
)

type wailsDesktopController struct {
	lastOverlayActive       bool
	lastOverlaySkip         bool
	lastOverlayLang         string
	lastOverlayText         string
	lastOverlayTheme        string
	lastOverlaySessionStart time.Time
	overlayFailureLogged    bool
	lastLanguage            string
	reminderActionOrder     []int64
	reminderOrderMu         sync.RWMutex
	lastStatusBarStatus     string
	lastStatusBarCountdown  string
	lastStatusBarTitle      string
	lastStatusBarPaused     bool
	lastRemindersPayload    string
	lastDetailsVisible      bool
	hasStatusBarSnapshot    bool
	statusBarSyncMu         sync.Mutex
	statusBarDetailsVisible atomic.Bool
	statusBar               desktop.StatusBarController
	overlay                 desktop.BreakOverlayController
	startOnce               sync.Once
}

func newDesktopController() desktopController {
	controller := &wailsDesktopController{
		statusBar: desktop.NewStatusBarController(),
		overlay:   desktop.NewBreakOverlayController(),
	}
	controller.statusBarDetailsVisible.Store(true)
	return controller
}

func (c *wailsDesktopController) OnStartup(ctx context.Context, app *App) {
	c.startOnce.Do(func() {
		logx.SetSink(func(level logx.Level, message string) {
			switch level {
			case logx.LevelError:
				runtime.LogError(ctx, message)
			case logx.LevelWarn:
				runtime.LogWarning(ctx, message)
			case logx.LevelInfo:
				runtime.LogInfo(ctx, message)
			default:
				runtime.LogDebug(ctx, message)
			}
		})
		go func() {
			<-ctx.Done()
			shutdownPreferredThemeProvider()
			logx.ClearSink()
		}()

		initPreferredThemeProvider()
		desktop.ConfigureDesktopWindowBehavior()
		c.statusBar.Init(func(event desktop.StatusBarEvent) {
			c.handleStatusBarEvent(ctx, app, event)
		})
		c.overlay.Init(func() {
			skipMode := overlaySkipMode(app.engine.GetSettings())
			_, err := app.skipCurrentBreakWithMode(skipMode)
			c.logErr(ctx, err)
		})
		settings := app.engine.GetSettings()
		c.lastLanguage = resolveEffectiveLanguage(settings.UI.Language)
		c.statusBar.SetLocale(buildStatusBarLocaleStrings(c.lastLanguage))
		go c.runtimeLoop(ctx, app)
	})
}

func (c *wailsDesktopController) runtimeLoop(ctx context.Context, app *App) {
	const statusLoopOffset = 150 * time.Millisecond
	offsetTimer := time.NewTimer(statusLoopOffset)
	defer offsetTimer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-offsetTimer.C:
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	defer c.statusBar.Destroy()
	defer c.overlay.Destroy()

	settings := app.engine.GetSettings()
	state := app.engine.GetRuntimeState(time.Now())
	c.syncStatusBarWithLock(state, settings)
	c.syncOverlay(ctx, state, settings)

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			settings := app.engine.GetSettings()
			state := app.engine.GetRuntimeState(now)
			c.syncStatusBarWithLock(state, settings)
			c.syncOverlay(ctx, state, settings)
		}
	}
}

func (c *wailsDesktopController) logErr(_ context.Context, err error) {
	if err == nil {
		return
	}
	logx.Errorf("desktop.error err=%v", err)
}
