package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"pause/internal/core/config"
	"pause/internal/core/history"
	"pause/internal/core/service"
	"pause/internal/logx"
	"pause/internal/meta"
	"pause/internal/paths"
	"pause/internal/platform"
)

type App struct {
	ctx           context.Context
	engine        *service.Engine
	history       *history.Store
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
	historyPath := defaultHistoryPath(configPath)
	historyStore, err := history.OpenStore(historyPath)
	if err != nil {
		return nil, err
	}

	adapters := platform.NewAdapters(meta.EffectiveAppBundleID())
	engine := service.NewEngine(
		store,
		adapters.IdleProvider,
		adapters.SoundPlayer,
		adapters.StartupManager,
		historyStore,
	)
	if err := syncEngineRemindersFromHistory(engine, historyStore); err != nil {
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
		engine:   engine,
		history:  historyStore,
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

func (a *App) GetReminders() ([]config.ReminderConfig, error) {
	if a == nil || a.history == nil {
		return nil, errors.New("history store unavailable")
	}
	defs, err := a.history.ListReminders()
	if err != nil {
		return nil, err
	}
	return historyDefsToConfig(defs), nil
}

func (a *App) UpdateReminders(patches []config.ReminderPatch) ([]config.ReminderConfig, error) {
	if a == nil || a.history == nil {
		return nil, errors.New("history store unavailable")
	}
	patchesJSON := marshalReminderPatchesForLog(patches)
	if err := applyReminderPatchToHistory(a.history, patches); err != nil {
		logx.Warnf("app.update_reminders_err stage=history_apply patches=%s err=%v", patchesJSON, err)
		return nil, err
	}
	reminders, err := loadReminderConfigsFromHistory(a.history)
	if err != nil {
		logx.Warnf("app.update_reminders_err stage=history_reload patches=%s err=%v", patchesJSON, err)
		return nil, err
	}
	a.engine.SetReminderConfigs(reminders)
	logx.Infof("app.reminders_updated patches=%s count=%d", patchesJSON, len(reminders))
	return reminders, nil
}

func (a *App) GetRuntimeState() config.RuntimeState {
	state := a.engine.GetRuntimeState(time.Now())
	return a.decorateRuntimeState(state)
}

func (a *App) Pause() (config.RuntimeState, error) {
	state, err := a.engine.Pause(time.Now())
	if err != nil {
		return config.RuntimeState{}, err
	}
	return a.decorateRuntimeState(state), nil
}

func (a *App) Resume() config.RuntimeState {
	return a.decorateRuntimeState(a.engine.Resume(time.Now()))
}

func (a *App) PauseReminder(reason string) (config.RuntimeState, error) {
	state, err := a.engine.PauseReminder(reason, time.Now())
	if err != nil {
		return config.RuntimeState{}, err
	}
	return a.decorateRuntimeState(state), nil
}

func (a *App) ResumeReminder(reason string) (config.RuntimeState, error) {
	state, err := a.engine.ResumeReminder(reason, time.Now())
	if err != nil {
		return config.RuntimeState{}, err
	}
	return a.decorateRuntimeState(state), nil
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
		default:
			cleaned := strings.TrimSpace(reason)
			if cleaned != "" {
				runes := []rune(cleaned)
				runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
				parts = append(parts, string(runes))
			}
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

func defaultHistoryPath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "history.db")
}

func historyDefsToConfig(defs []history.ReminderDefinition) []config.ReminderConfig {
	result := make([]config.ReminderConfig, 0, len(defs))
	for _, def := range defs {
		id := strings.ToLower(strings.TrimSpace(def.ID))
		if id == "" {
			continue
		}
		result = append(result, config.ReminderConfig{
			ID:           id,
			Name:         strings.TrimSpace(def.Name),
			Enabled:      def.Enabled,
			IntervalSec:  def.IntervalSec,
			BreakSec:     def.BreakSec,
			DeliveryType: strings.TrimSpace(def.DeliveryType),
		})
	}
	return config.NormalizeReminderConfigs(result)
}

func loadReminderConfigsFromHistory(store *history.Store) ([]config.ReminderConfig, error) {
	if store == nil {
		return nil, nil
	}
	defs, err := store.ListReminders()
	if err != nil {
		return nil, err
	}
	return historyDefsToConfig(defs), nil
}

func syncEngineRemindersFromHistory(engine *service.Engine, store *history.Store) error {
	if engine == nil || store == nil {
		return nil
	}
	reminders, err := loadReminderConfigsFromHistory(store)
	if err != nil {
		return err
	}
	if len(reminders) == 0 {
		return nil
	}
	engine.SetReminderConfigs(reminders)
	logx.Infof("app.reminders_synced source=history count=%d", len(reminders))
	return nil
}

func applyReminderPatchToHistory(store *history.Store, patches []config.ReminderPatch) error {
	if store == nil || len(patches) == 0 {
		return nil
	}
	mutations := make([]history.ReminderMutation, 0, len(patches))
	for _, patch := range patches {
		id := strings.ToLower(strings.TrimSpace(patch.ID))
		if id == "" {
			continue
		}
		mutations = append(mutations, history.ReminderMutation{
			ID:           id,
			Name:         patch.Name,
			Enabled:      patch.Enabled,
			IntervalSec:  patch.IntervalSec,
			BreakSec:     patch.BreakSec,
			DeliveryType: patch.DeliveryType,
		})
	}
	return store.UpdateReminders(mutations)
}

func currentWeekRange(now time.Time) (time.Time, time.Time) {
	local := now.Local()
	weekday := int(local.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location()).
		AddDate(0, 0, -(weekday - 1))
	end := start.AddDate(0, 0, 7)
	return start, end
}

func marshalReminderPatchesForLog(patches []config.ReminderPatch) string {
	raw, err := json.Marshal(patches)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func (a *App) GetReminderWeeklyStats(weekStartSec int64, weekEndSec int64) (history.WeeklyStats, error) {
	if a == nil || a.history == nil {
		return history.WeeklyStats{}, errors.New("history store unavailable")
	}

	var weekStart time.Time
	var weekEnd time.Time
	if weekStartSec == 0 && weekEndSec == 0 {
		weekStart, weekEnd = currentWeekRange(time.Now())
	} else {
		if weekEndSec <= weekStartSec {
			return history.WeeklyStats{}, errors.New("invalid time range")
		}
		weekStart = time.Unix(weekStartSec, 0)
		weekEnd = time.Unix(weekEndSec, 0)
	}
	return a.history.QueryWeeklyStats(weekStart, weekEnd)
}

func (a *App) Shutdown(_ context.Context) {
	if a == nil || a.history == nil {
		return
	}
	if err := a.history.Close(); err != nil {
		logx.Warnf("app.shutdown history_close_err=%v", err)
	}
}

func (a *App) decorateRuntimeState(state config.RuntimeState) config.RuntimeState {
	settings := a.engine.GetSettings()
	state.EffectiveLanguage = resolveEffectiveLanguage(settings.UI.Language)
	state.EffectiveTheme = resolveEffectiveTheme(settings.UI.Theme)
	return decorateRuntimeStateForPlatform(state)
}
