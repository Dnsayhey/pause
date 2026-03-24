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
	"pause/internal/core/reminder"
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
	if store.WasCreated() {
		language := resolveEffectiveLanguage(store.Get().UI.Language)
		if err := ensureBuiltInRemindersForFirstInstall(historyStore, language); err != nil {
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
	reminders, err := reloadAndSyncReminders(a.engine, a.history)
	if err != nil {
		logx.Warnf("app.update_reminders_err stage=history_reload patches=%s err=%v", patchesJSON, err)
		return nil, err
	}
	logx.Infof("app.reminders_updated patches=%s count=%d", patchesJSON, len(reminders))
	return reminders, nil
}

func (a *App) CreateReminder(input config.ReminderCreateInput) ([]config.ReminderConfig, error) {
	if a == nil || a.history == nil {
		return nil, errors.New("history store unavailable")
	}

	normalized, err := normalizeReminderCreateInput(input)
	if err != nil {
		return nil, err
	}

	id, err := createReminderInHistory(a.history, normalized)
	if err != nil {
		logx.Warnf("app.create_reminder_err stage=history_create id=%d err=%v", id, err)
		return nil, err
	}
	reminders, err := reloadAndSyncReminders(a.engine, a.history)
	if err != nil {
		logx.Warnf("app.create_reminder_err stage=history_reload id=%d err=%v", id, err)
		return nil, err
	}
	logx.Infof("app.reminder_created id=%d count=%d", id, len(reminders))
	return reminders, nil
}

func (a *App) DeleteReminder(reminderID int64) ([]config.ReminderConfig, error) {
	if a == nil || a.history == nil {
		return nil, errors.New("history store unavailable")
	}
	id := reminderID
	if id <= 0 {
		return nil, errors.New("reminder id is required")
	}
	if err := deleteReminderInHistory(a.history, id); err != nil {
		logx.Warnf("app.delete_reminder_err stage=history_delete id=%d err=%v", id, err)
		return nil, err
	}
	reminders, err := reloadAndSyncReminders(a.engine, a.history)
	if err != nil {
		logx.Warnf("app.delete_reminder_err stage=history_reload id=%d err=%v", id, err)
		return nil, err
	}
	logx.Infof("app.reminder_deleted id=%d count=%d", id, len(reminders))
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

func (a *App) PauseReminder(reminderID int64) (config.RuntimeState, error) {
	if reminderID <= 0 {
		return config.RuntimeState{}, errors.New("reminder id is required")
	}
	state, err := a.engine.PauseReminder(reminderID, time.Now())
	if err != nil {
		return config.RuntimeState{}, err
	}
	return a.decorateRuntimeState(state), nil
}

func (a *App) ResumeReminder(reminderID int64) (config.RuntimeState, error) {
	if reminderID <= 0 {
		return config.RuntimeState{}, errors.New("reminder id is required")
	}
	state, err := a.engine.ResumeReminder(reminderID, time.Now())
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

func (a *App) StartBreakNowForReason(reminderID int64) (config.RuntimeState, error) {
	if reminderID < 0 {
		return config.RuntimeState{}, errors.New("reminder id is invalid")
	}
	state, err := a.engine.StartBreakNowForReason(reminderID, time.Now())
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

func buildBreakNotificationBody(state config.RuntimeState) string {
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

func historyDefsToConfig(defs []history.ReminderDefinition) []config.ReminderConfig {
	result := make([]config.ReminderConfig, 0, len(defs))
	for _, def := range defs {
		id := def.ID
		if id <= 0 {
			continue
		}
		result = append(result, config.ReminderConfig{
			ID:           id,
			Name:         strings.TrimSpace(def.Name),
			Enabled:      def.Enabled,
			IntervalSec:  def.IntervalSec,
			BreakSec:     def.BreakSec,
			ReminderType: strings.TrimSpace(def.ReminderType),
		})
	}
	return reminder.CloneConfigs(result)
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
	reminders, err := reloadAndSyncReminders(engine, store)
	if err != nil {
		return err
	}
	logx.Infof("app.reminders_synced source=history count=%d", len(reminders))
	return nil
}

func reloadAndSyncReminders(engine *service.Engine, store *history.Store) ([]config.ReminderConfig, error) {
	if engine == nil || store == nil {
		return nil, nil
	}
	reminders, err := loadReminderConfigsFromHistory(store)
	if err != nil {
		return nil, err
	}
	engine.SetReminderConfigs(reminders)
	return reminders, nil
}

func applyReminderPatchToHistory(store *history.Store, patches []config.ReminderPatch) error {
	if store == nil || len(patches) == 0 {
		return nil
	}

	existingDefs, err := store.ListReminders()
	if err != nil {
		return err
	}
	existingByID := make(map[int64]struct{}, len(existingDefs))
	for _, def := range existingDefs {
		id := def.ID
		if id <= 0 {
			continue
		}
		existingByID[id] = struct{}{}
	}

	mutations := make([]history.ReminderMutation, 0, len(patches))
	for _, patch := range patches {
		id := patch.ID
		if id <= 0 {
			return errors.New("reminder id is required")
		}
		if _, ok := existingByID[id]; !ok {
			return fmt.Errorf("reminder id %d not found", id)
		}

		mutation := history.ReminderMutation{ID: id}
		if patch.Name != nil {
			if err := validateReminderNameInput(*patch.Name); err != nil {
				return err
			}
			name := *patch.Name
			if name == "" {
				return errors.New("reminder name is required")
			}
			mutation.Name = &name
		}
		if patch.Enabled != nil {
			mutation.Enabled = patch.Enabled
		}
		if patch.IntervalSec != nil {
			if *patch.IntervalSec <= 0 {
				return errors.New("reminder intervalSec must be > 0")
			}
			intervalSec := *patch.IntervalSec
			mutation.IntervalSec = &intervalSec
		}
		if patch.BreakSec != nil {
			if *patch.BreakSec <= 0 {
				return errors.New("reminder breakSec must be > 0")
			}
			breakSec := *patch.BreakSec
			mutation.BreakSec = &breakSec
		}
		if patch.ReminderType != nil {
			if err := validateReminderTypeInput(*patch.ReminderType); err != nil {
				return err
			}
			reminderType := *patch.ReminderType
			mutation.ReminderType = &reminderType
		}
		mutations = append(mutations, mutation)
	}
	return store.UpdateReminders(mutations)
}

func normalizeReminderCreateInput(input config.ReminderCreateInput) (config.ReminderCreateInput, error) {
	if err := validateReminderNameInput(input.Name); err != nil {
		return config.ReminderCreateInput{}, err
	}
	if input.IntervalSec <= 0 {
		return config.ReminderCreateInput{}, errors.New("reminder intervalSec must be > 0")
	}
	if input.BreakSec <= 0 {
		return config.ReminderCreateInput{}, errors.New("reminder breakSec must be > 0")
	}
	if input.ReminderType == nil {
		return config.ReminderCreateInput{}, errors.New("reminder reminderType is required")
	}
	if err := validateReminderTypeInput(*input.ReminderType); err != nil {
		return config.ReminderCreateInput{}, err
	}
	return input, nil
}

func validateReminderNameInput(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("reminder name is required")
	}
	if name != strings.TrimSpace(name) {
		return errors.New("reminder name cannot have leading or trailing spaces")
	}
	return nil
}

func validateReminderTypeInput(value string) error {
	if value != "rest" && value != "notify" {
		return errors.New("reminder reminderType must be rest or notify")
	}
	return nil
}

func createReminderInHistory(store *history.Store, input config.ReminderCreateInput) (int64, error) {
	if store == nil {
		return 0, nil
	}

	name := input.Name
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	intervalSec := input.IntervalSec
	breakSec := input.BreakSec

	return store.CreateReminder(history.ReminderMutation{
		Name:         &name,
		Enabled:      &enabled,
		IntervalSec:  &intervalSec,
		BreakSec:     &breakSec,
		ReminderType: input.ReminderType,
	})
}

func deleteReminderInHistory(store *history.Store, reminderID int64) error {
	if store == nil {
		return nil
	}
	return store.DeleteReminder(reminderID)
}

func ensureBuiltInRemindersForFirstInstall(store *history.Store, language string) error {
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
	reminders := []history.ReminderMutation{
		{
			Name:         &eyeName,
			Enabled:      &eyeEnabled,
			IntervalSec:  &eyeIntervalSec,
			BreakSec:     &eyeBreakSec,
			ReminderType: &restType,
		},
		{
			Name:         &standName,
			Enabled:      &standEnabled,
			IntervalSec:  &standIntervalSec,
			BreakSec:     &standBreakSec,
			ReminderType: &restType,
		},
		{
			Name:         &waterName,
			Enabled:      &waterEnabled,
			IntervalSec:  &waterIntervalSec,
			BreakSec:     &waterBreakSec,
			ReminderType: &notifyType,
		},
	}

	for _, reminder := range reminders {
		if _, err := store.CreateReminder(reminder); err != nil && !errors.Is(err, history.ErrReminderAlreadyExists) {
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

func marshalReminderPatchesForLog(patches []config.ReminderPatch) string {
	raw, err := json.Marshal(patches)
	if err != nil {
		return "[]"
	}
	return string(raw)
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

func (a *App) GetAnalyticsHourlyHeatmap(fromSec int64, toSec int64, metric string) (history.AnalyticsHourlyHeatmap, error) {
	if a == nil || a.history == nil {
		return history.AnalyticsHourlyHeatmap{}, errors.New("history store unavailable")
	}
	from, to, err := resolveAnalyticsRange(fromSec, toSec)
	if err != nil {
		return history.AnalyticsHourlyHeatmap{}, err
	}
	return a.history.QueryAnalyticsHourlyHeatmap(from, to, history.AnalyticsHeatmapMetric(strings.TrimSpace(metric)))
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

func (a *App) decorateRuntimeState(state config.RuntimeState) config.RuntimeState {
	settings := a.engine.GetSettings()
	state.EffectiveLanguage = resolveEffectiveLanguage(settings.UI.Language)
	state.EffectiveTheme = resolveEffectiveTheme(settings.UI.Theme)
	return decorateRuntimeStateForPlatform(state)
}
