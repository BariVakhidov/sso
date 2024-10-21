package models

import "github.com/google/uuid"

type User struct {
	ID       uuid.UUID
	Email    string
	PassHash []byte
}

type UserEvent struct {
	ID    uuid.UUID
	Email string
}
