package model

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID         uuid.UUID    `db:"id"`
	Type       string       `db:"event_type"`
	Payload    string       `db:"payload"`
	Status     string       `db:"status"`
	CreatedAt  time.Time    `db:"created_at"`
	ReservedTo sql.NullTime `db:"reserved_to"`
}
