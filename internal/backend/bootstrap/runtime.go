package bootstrap

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	historyadapter "pause/internal/backend/adapters/history"
	settingsadapter "pause/internal/backend/adapters/settings"
	"pause/internal/backend/ports"
	service "pause/internal/backend/runtime/engine"
	"pause/internal/backend/storage/historydb"
	"pause/internal/backend/storage/settingsjson"
	analyticsusecase "pause/internal/backend/usecase/analytics"
	reminderusecase "pause/internal/backend/usecase/reminder"
	settingsusecase "pause/internal/backend/usecase/settings"
	"pause/internal/logx"
	"pause/internal/platform"
)

type Runtime struct {
	Settings                       *settingsjson.Store
	History                        *historydb.Store
	HistoryPath                    string
	Engine                         RuntimeEngine
	ReminderService                *reminderusecase.Service
	AnalyticsService               *analyticsusecase.Service
	SettingsService                *settingsusecase.Service
	Notifier                       ports.Notifier
	NotificationCapabilityProvider ports.NotificationCapabilityProvider
}

func NewRuntime(configPath string, bundleID string) (*Runtime, error) {
	cleanPath := strings.TrimSpace(configPath)
	if cleanPath == "" {
		return nil, errors.New("config path is required")
	}

	store, err := settingsjson.OpenStore(cleanPath)
	if err != nil {
		return nil, err
	}
	historyPath := defaultHistoryPath(cleanPath)
	historyStore, err := historydb.OpenStore(context.Background(), historyPath)
	if err != nil {
		return nil, err
	}
	container, err := NewContainer(historyStore)
	if err != nil {
		closeHistoryStoreOnInitError(historyStore, "new_container")
		return nil, err
	}

	adapters := platform.NewAdapters(bundleID)
	breakRecorder := historyadapter.NewBreakRecorder(historyStore)
	engine := service.NewEngine(
		store,
		adapters.IdleProvider,
		adapters.LockStateProvider,
		adapters.SoundPlayer,
		adapters.Notifier,
		breakRecorder,
	)
	container.ReminderService.SetRuntimeSink(engine)
	if err := container.ReminderService.Sync(context.Background()); err != nil {
		closeHistoryStoreOnInitError(historyStore, "reminder_sync")
		return nil, err
	}

	settingsRepo := settingsadapter.NewSettingsRepository(store, adapters.StartupManager)
	settingsService, err := settingsusecase.NewService(settingsRepo)
	if err != nil {
		closeHistoryStoreOnInitError(historyStore, "settings_service")
		return nil, err
	}

	return &Runtime{
		Settings:                       store,
		History:                        historyStore,
		HistoryPath:                    historyPath,
		Engine:                         WrapEngine(engine),
		ReminderService:                container.ReminderService,
		AnalyticsService:               container.AnalyticsService,
		SettingsService:                settingsService,
		Notifier:                       adapters.Notifier,
		NotificationCapabilityProvider: adapters.NotificationCapabilityProvider,
	}, nil
}

func (r *Runtime) Close() error {
	if r == nil || r.History == nil {
		return nil
	}
	return r.History.Close()
}

func defaultHistoryPath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "history.db")
}

func closeHistoryStoreOnInitError(historyStore *historydb.Store, stage string) {
	if historyStore == nil {
		return
	}
	if err := historyStore.Close(); err != nil {
		logx.Warnf("runtime.init cleanup_close_err stage=%s err=%v", stage, err)
	}
}
