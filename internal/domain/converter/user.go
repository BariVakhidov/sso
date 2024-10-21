package converter

import (
	"github.com/BariVakhidov/sso/internal/domain/models"
	storageModel "github.com/BariVakhidov/sso/internal/storage/model"
)

func ToUserFromStorage(storageUser storageModel.User) models.User {
	return models.User{
		ID:       storageUser.ID,
		Email:    storageUser.Email,
		PassHash: storageUser.PassHash,
	}
}

func ToUserEventFromStorage(storageUser storageModel.User) models.UserEvent {
	return models.UserEvent{
		ID:    storageUser.ID,
		Email: storageUser.Email,
	}
}
