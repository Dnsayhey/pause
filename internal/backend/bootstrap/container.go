package bootstrap

import (
	"errors"

	historyadapter "pause/internal/backend/adapters/history"
	settingsadapter "pause/internal/backend/adapters/settings"
	runtimeengine "pause/internal/backend/runtime/engine"
	historydb "pause/internal/backend/storage/historydb"
	analyticsusecase "pause/internal/backend/usecase/analytics"
	reminderusecase "pause/internal/backend/usecase/reminder"
	settingsusecase "pause/internal/backend/usecase/settings"
)

type Container struct {
	ReminderService  *reminderusecase.Service
	AnalyticsService *analyticsusecase.Service
}

func NewContainer(historyStore *historydb.Store) (*Container, error) {
	if historyStore == nil {
		return nil, errors.New("history store unavailable")
	}
	reminderRepo := historyadapter.NewReminderRepository(historyStore)
	reminderService, err := reminderusecase.NewService(reminderRepo)
	if err != nil {
		return nil, err
	}
	analyticsRepo := historyadapter.NewAnalyticsRepository(historyStore)
	analyticsService, err := analyticsusecase.NewService(analyticsRepo)
	if err != nil {
		return nil, err
	}
	return &Container{
		ReminderService:  reminderService,
		AnalyticsService: analyticsService,
	}, nil
}

func NewSettingsService(engine *runtimeengine.Engine) (*settingsusecase.Service, error) {
	if engine == nil {
		return nil, errors.New("engine unavailable")
	}
	settingsRepo := settingsadapter.NewSettingsRepository(engine)
	return settingsusecase.NewService(settingsRepo)
}
