package models

import "time"

type FailedLogin struct {
	Attempts    int
	FirstFail   time.Time
	LockedUntil time.Time
}
