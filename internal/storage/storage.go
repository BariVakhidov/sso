package storage

import "errors"

var (
	ErrUserExists          = errors.New("user already exists")
	ErrAppExists           = errors.New("app already exists")
	ErrUserNotFound        = errors.New("user not found")
	ErrAppNotFound         = errors.New("app not found")
	ErrFailedLoginNotFound = errors.New("failed login not found")
)
