package auth

const (
	ErrInvalidEmail           = "invalid email format"
	ErrPasswordRequired       = "password is required"
	ErrEmailRequired          = "email is required"
	ErrUserIDRequired         = "userID is required"
	ErrUserNotFound           = "user not found"
	ErrUserExists             = "user already exists"
	ErrAppExists              = "app already exists"
	ErrAppNotFound            = "app not found"
	ErrAppNameRequired        = "app name required"
	ErrAppSecretRequired      = "app secret required"
	ErrAppIDRequired          = "app_id is required"
	ErrInternal               = "internal error"
	ErrInvalidCredentials     = "invalid credentials"
	ErrAccountTemporaryLocked = "account is temporary locked"
)
