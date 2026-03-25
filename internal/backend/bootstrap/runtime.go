package bootstrap

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	historyadapter "pause/internal/backend/adapters/history"
	"pause/internal/backend/storage/historydb"
	"pause/internal/backend/storage/settingsjson"
	analyticsusecase "pause/internal/backend/usecase/analytics"
	reminderusecase "pause/internal/backend/usecase/reminder"
	settingsusecase "pause/internal/backend/usecase/settings"
	"pause/internal/core/service"
	"pause/internal/platform"
)

type Runtime struct {
	Settings         *settingsjson.Store
	History          *historydb.Store
	HistoryPath      string
	Engine           *service.Engine
	ReminderService  *reminderusecase.Service
	AnalyticsService *analyticsusecase.Service
	SettingsService  *settingsusecase.Service
	Notifier         platform.Notifier
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
		_ = historyStore.Close()
		return nil, err
	}

	adapters := platform.NewAdapters(bundleID)
	breakRecorder := historyadapter.NewBreakRecorder(historyStore)
	engine := service.NewEngine(
		store,
		adapters.IdleProvider,
		adapters.LockStateProvider,
		adapters.SoundPlayer,
		adapters.StartupManager,
		breakRecorder,
	)
	engine.SetNotifier(adapters.Notifier)

	settingsService, err := NewSettingsService(engine)
	if err != nil {
		_ = historyStore.Close()
		return nil, err
	}

	return &Runtime{
		Settings:         store,
		History:          historyStore,
		HistoryPath:      historyPath,
		Engine:           engine,
		ReminderService:  container.ReminderService,
		AnalyticsService: container.AnalyticsService,
		SettingsService:  settingsService,
		Notifier:         adapters.Notifier,
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
