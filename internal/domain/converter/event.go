package converter

import (
	"github.com/BariVakhidov/sso/internal/domain/models"
	storageModel "github.com/BariVakhidov/sso/internal/storage/model"
)

func ToEventFromStorage(storageEvent storageModel.Event) models.Event {
	return models.Event{
		ID:      storageEvent.ID,
		Type:    storageEvent.Type,
		Payload: storageEvent.Payload,
	}
}

func ToEventsFromStorage(storageEvents []storageModel.Event) []models.Event {
	events := make([]models.Event, len(storageEvents))
	for i, event := range storageEvents {
		events[i] = ToEventFromStorage(event)
	}

	return events
}
