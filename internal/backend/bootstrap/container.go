package bootstrap

import (
	"errors"

	historyadapter "pause/internal/backend/adapters/history"
	reminderusecase "pause/internal/backend/usecase/reminder"
	corehistory "pause/internal/core/history"
)

type Container struct {
	ReminderService *reminderusecase.Service
}

func NewContainer(historyStore *corehistory.HistoryStore) (*Container, error) {
	if historyStore == nil {
		return nil, errors.New("history store unavailable")
	}
	reminderRepo := historyadapter.NewReminderRepository(historyStore)
	reminderService, err := reminderusecase.NewService(reminderRepo)
	if err != nil {
		return nil, err
	}
	return &Container{
		ReminderService: reminderService,
	}, nil
}
