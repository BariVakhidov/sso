package model

import "github.com/google/uuid"

type User struct {
	ID       uuid.UUID `db:"id"`
	Email    string    `db:"email"`
	PassHash []byte    `db:"pass_hash"`
}
