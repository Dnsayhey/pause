package bootstrap

import (
	"errors"

	historyadapter "pause/internal/backend/adapters/history"
	"pause/internal/backend/ports"
	historydb "pause/internal/backend/storage/historydb"
	analyticsusecase "pause/internal/backend/usecase/analytics"
	reminderusecase "pause/internal/backend/usecase/reminder"
)

type Container struct {
	ReminderService  *reminderusecase.Service
	AnalyticsService *analyticsusecase.Service
}

func NewContainer(historyStore *historydb.Store, reminderSink ports.ReminderRuntimeSink) (*Container, error) {
	if historyStore == nil {
		return nil, errors.New("history store unavailable")
	}
	reminderRepo := historyadapter.NewReminderRepository(historyStore)
	reminderService, err := reminderusecase.NewService(reminderRepo, reminderSink)
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
