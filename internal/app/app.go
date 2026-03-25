package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"pause/internal/backend/bootstrap"
	reminderdomain "pause/internal/backend/domain/reminder"
	"pause/internal/core/history"
	"pause/internal/core/reminder"
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

	logx.Infof(
		"app.init bundle_id=%s config_path=%s history_path=%s config_created=%t",
		meta.EffectiveAppBundleID(),
		configPath,
		historyPath,
		store.WasCreated(),
	)

	return &App{
		engine:    engine,
		history:   historyStore,
		reminders: container.ReminderService,
		notifier:  adapters.Notifier,
		desktop:   newDesktopController(),
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

func (a *App) GetSettings() settings.Settings {
	return a.engine.GetSettings()
}

func (a *App) UpdateSettings(patch settings.SettingsPatch) (settings.Settings, error) {
	nextSettings, err := a.engine.UpdateSettings(patch)
	if err != nil {
		logx.Warnf("app.update_settings_err err=%v", err)
		return settings.Settings{}, err
	}
	return nextSettings, nil
}

func (a *App) GetReminders() ([]reminder.ReminderConfig, error) {
	if a == nil || a.reminders == nil {
		return nil, errors.New("reminder service unavailable")
	}
	defs, err := a.reminders.List(appContextOrBackground(a.ctx))
	if err != nil {
		return nil, err
	}
	return reminderDefsToConfig(defs), nil
}

func (a *App) UpdateReminder(patch reminder.ReminderPatch) ([]reminder.ReminderConfig, error) {
	if a == nil || a.reminders == nil {
		return nil, errors.New("reminder service unavailable")
	}
	ctx := appContextOrBackground(a.ctx)
	defs, err := a.reminders.Update(ctx, reminderdomain.Patch{
		ID:           patch.ID,
		Name:         patch.Name,
		Enabled:      patch.Enabled,
		IntervalSec:  patch.IntervalSec,
		BreakSec:     patch.BreakSec,
		ReminderType: patch.ReminderType,
	})
	if err != nil {
		logx.Warnf("app.update_reminder_err stage=usecase_update patch_id=%d err=%v", patch.ID, err)
		return nil, err
	}
	reminders := reminderDefsToConfig(defs)
	a.engine.SetReminderConfigs(reminders)
	logx.Infof("app.reminder_updated id=%d count=%d", patch.ID, len(reminders))
	return reminders, nil
}

func (a *App) CreateReminder(input reminder.ReminderCreateInput) ([]reminder.ReminderConfig, error) {
	if a == nil || a.reminders == nil {
		return nil, errors.New("reminder service unavailable")
	}

	ctx := appContextOrBackground(a.ctx)
	defs, err := a.reminders.Create(ctx, reminderdomain.CreateInput{
		Name:         input.Name,
		IntervalSec:  input.IntervalSec,
		BreakSec:     input.BreakSec,
		Enabled:      input.Enabled,
		ReminderType: input.ReminderType,
	})
	if err != nil {
		logx.Warnf("app.create_reminder_err stage=usecase_create err=%v", err)
		return nil, err
	}
	reminders := reminderDefsToConfig(defs)
	a.engine.SetReminderConfigs(reminders)
	createdID := int64(0)
	if len(reminders) > 0 {
		createdID = reminders[len(reminders)-1].ID
	}
	logx.Infof("app.reminder_created id=%d count=%d", createdID, len(reminders))
	return reminders, nil
}

func (a *App) DeleteReminder(reminderID int64) ([]reminder.ReminderConfig, error) {
	if a == nil || a.reminders == nil {
		return nil, errors.New("reminder service unavailable")
	}
	id := reminderID
	ctx := appContextOrBackground(a.ctx)
	defs, err := a.reminders.Delete(ctx, id)
	if err != nil {
		logx.Warnf("app.delete_reminder_err stage=usecase_delete id=%d err=%v", id, err)
		return nil, err
	}
	reminders := reminderDefsToConfig(defs)
	a.engine.SetReminderConfigs(reminders)
	logx.Infof("app.reminder_deleted id=%d count=%d", id, len(reminders))
	return reminders, nil
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

func (a *App) GetLaunchAtLogin() (bool, error) {
	return a.engine.GetLaunchAtLogin()
}

func (a *App) SetLaunchAtLogin(enabled bool) (bool, error) {
	return a.engine.SetLaunchAtLogin(enabled)
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

func reminderDefsToConfig(defs []reminderdomain.Reminder) []reminder.ReminderConfig {
	result := make([]reminder.ReminderConfig, 0, len(defs))
	for _, def := range defs {
		id := def.ID
		if id <= 0 {
			continue
		}
		result = append(result, reminder.ReminderConfig{
			ID:           id,
			Name:         strings.TrimSpace(def.Name),
			Enabled:      def.Enabled,
			IntervalSec:  def.IntervalSec,
			BreakSec:     def.BreakSec,
			ReminderType: strings.TrimSpace(def.ReminderType),
		})
	}
	return cloneReminderConfigs(result)
}

func historyDefsToConfig(defs []history.Reminder) []reminder.ReminderConfig {
	result := make([]reminder.ReminderConfig, 0, len(defs))
	for _, def := range defs {
		id := def.ID
		if id <= 0 {
			continue
		}
		result = append(result, reminder.ReminderConfig{
			ID:           id,
			Name:         strings.TrimSpace(def.Name),
			Enabled:      def.Enabled,
			IntervalSec:  def.IntervalSec,
			BreakSec:     def.BreakSec,
			ReminderType: strings.TrimSpace(def.ReminderType),
		})
	}
	return cloneReminderConfigs(result)
}

func cloneReminderConfigs(reminders []reminder.ReminderConfig) []reminder.ReminderConfig {
	if len(reminders) == 0 {
		return nil
	}
	cloned := make([]reminder.ReminderConfig, 0, len(reminders))
	cloned = append(cloned, reminders...)
	return cloned
}

func ensureBuiltInRemindersForFirstInstall(ctx context.Context, store *history.HistoryStore, language string) error {
	if store == nil {
		return nil
	}

	eyeName, standName, waterName := localizedBuiltInReminderSeedNames(language)
	eyeEnabled := true
	standEnabled := false
	waterEnabled := false
	restType := "rest"
	notifyType := "notify"
	eyeIntervalSec := 20 * 60
	eyeBreakSec := 20
	standIntervalSec := 60 * 60
	standBreakSec := 5 * 60
	waterIntervalSec := 45 * 60
	waterBreakSec := 1
	reminders := []history.Reminder{
		{
			Name:         eyeName,
			Enabled:      eyeEnabled,
			IntervalSec:  eyeIntervalSec,
			BreakSec:     eyeBreakSec,
			ReminderType: restType,
		},
		{
			Name:         standName,
			Enabled:      standEnabled,
			IntervalSec:  standIntervalSec,
			BreakSec:     standBreakSec,
			ReminderType: restType,
		},
		{
			Name:         waterName,
			Enabled:      waterEnabled,
			IntervalSec:  waterIntervalSec,
			BreakSec:     waterBreakSec,
			ReminderType: notifyType,
		},
	}

	for _, reminder := range reminders {
		if _, err := store.CreateReminder(appContextOrBackground(ctx), reminder); err != nil && !errors.Is(err, history.ErrReminderAlreadyExists) {
			return err
		}
	}
	return nil
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

func (a *App) GetAnalyticsWeeklyStats(fromSec int64, toSec int64) (history.AnalyticsWeeklyStats, error) {
	if a == nil || a.history == nil {
		return history.AnalyticsWeeklyStats{}, errors.New("history store unavailable")
	}
	from, to, err := resolveAnalyticsRange(fromSec, toSec)
	if err != nil {
		return history.AnalyticsWeeklyStats{}, err
	}
	return a.history.QueryAnalyticsWeeklyStats(from, to)
}

func (a *App) GetAnalyticsSummary(fromSec int64, toSec int64) (history.AnalyticsSummary, error) {
	if a == nil || a.history == nil {
		return history.AnalyticsSummary{}, errors.New("history store unavailable")
	}
	from, to, err := resolveAnalyticsRange(fromSec, toSec)
	if err != nil {
		return history.AnalyticsSummary{}, err
	}
	return a.history.QueryAnalyticsSummary(from, to)
}

func (a *App) GetAnalyticsTrendByDay(fromSec int64, toSec int64) (history.AnalyticsTrend, error) {
	if a == nil || a.history == nil {
		return history.AnalyticsTrend{}, errors.New("history store unavailable")
	}
	from, to, err := resolveAnalyticsRange(fromSec, toSec)
	if err != nil {
		return history.AnalyticsTrend{}, err
	}
	return a.history.QueryAnalyticsTrendByDay(from, to)
}

func (a *App) GetAnalyticsBreakTypeDistribution(fromSec int64, toSec int64) (history.AnalyticsBreakTypeDistribution, error) {
	if a == nil || a.history == nil {
		return history.AnalyticsBreakTypeDistribution{}, errors.New("history store unavailable")
	}
	from, to, err := resolveAnalyticsRange(fromSec, toSec)
	if err != nil {
		return history.AnalyticsBreakTypeDistribution{}, err
	}
	return a.history.QueryAnalyticsBreakTypeDistribution(from, to)
}

func resolveAnalyticsRange(fromSec int64, toSec int64) (time.Time, time.Time, error) {
	if fromSec == 0 && toSec == 0 {
		start, end := currentWeekRange(time.Now())
		return start, end, nil
	}
	if toSec <= fromSec {
		return time.Time{}, time.Time{}, errors.New("invalid time range")
	}
	return time.Unix(fromSec, 0), time.Unix(toSec, 0), nil
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
