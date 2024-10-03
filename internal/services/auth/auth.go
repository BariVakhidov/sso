package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/BariVakhidov/sso/internal/domain/models"
	"github.com/BariVakhidov/sso/internal/lib/jwt"
	"github.com/BariVakhidov/sso/internal/lib/logger/sl"
	"github.com/BariVakhidov/sso/internal/storage"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/peer"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidUserID      = errors.New("invalid userID")
	ErrUserExists         = errors.New("user exists")
	ErrAppExists          = errors.New("app exists")
	ErrAppNotFound        = errors.New("app not found")
	ErrUserNotFound       = errors.New("user not found")
	ErrAccountIsLocked    = errors.New("account is locked")
)

type Auth struct {
	log                  *slog.Logger
	userSaver            UserSaver
	userProvider         UserProvider
	appProvider          AppProvider
	tokenTTL             time.Duration
	failedLogins         *prometheus.CounterVec
	failedLoginsProvider FailedLoginProvider
}

type FailedLoginProvider interface {
	FailedLoginAttempts(ctx context.Context, userID uuid.UUID) (models.FailedLogin, error)
	SaveFailedLoginAttempts(ctx context.Context, userID uuid.UUID, failedLogin models.FailedLogin) error
	RemoveFailedLoginAttempts(ctx context.Context, userID uuid.UUID) error
}

type UserSaver interface {
	SaveUser(ctx context.Context, email string, passwordHash []byte) (user models.User, err error)
}

type UserProvider interface {
	User(ctx context.Context, email string) (models.User, error)
	IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error)
}

type AppProvider interface {
	App(ctx context.Context, appID uuid.UUID) (models.App, error)
	FindApp(ctx context.Context, name string) (models.App, error)
	CreateApp(ctx context.Context, name, secret string) (models.App, error)
}

const (
	MaxFailedLoginAttempts = 10
	attemptWindow          = 15 * time.Minute
	BaseLockoutDuration    = 15 * time.Second
)

// New returns a new instance of the Auth service
func New(
	log *slog.Logger,
	userSaver UserSaver,
	userProvider UserProvider,
	appProvider AppProvider,
	failedLoginsProvider FailedLoginProvider,
	tokenTTL time.Duration,
	failedLogins *prometheus.CounterVec,
) *Auth {
	return &Auth{
		log:                  log,
		userSaver:            userSaver,
		userProvider:         userProvider,
		appProvider:          appProvider,
		tokenTTL:             tokenTTL,
		failedLogins:         failedLogins,
		failedLoginsProvider: failedLoginsProvider,
	}
}

func (a *Auth) Login(ctx context.Context, email string, password string, appID uuid.UUID) (string, error) {
	const op = "auth.Login"
	log := a.log.With(
		slog.String("op", op),
		slog.String("username", email),
	)
	log.Info("attempting to login user")

	user, err := a.userProvider.User(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			a.log.Warn("user not found", sl.Err(err))
			return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}

		log.Error("failed to get user", sl.Err(err))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	// Check if user is currently locked out
	failedLoginAttempt, err := a.failedLoginsProvider.FailedLoginAttempts(ctx, user.ID)
	isFirstAttempt := errors.Is(err, storage.ErrFailedLoginNotFound)
	if err != nil && !isFirstAttempt {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	if !isFirstAttempt && time.Now().Before(failedLoginAttempt.LockedUntil) {
		log.Warn("account is locked", slog.String("email", email))
		return "", fmt.Errorf("%s: %w", op, ErrAccountIsLocked)
	}

	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		log.Error("invalid credentials", sl.Err(err))
		p, _ := peer.FromContext(ctx)
		a.failedLogins.WithLabelValues(email, p.Addr.String()).Inc()

		newAttempt := a.handleFailedLogin(user.ID, failedLoginAttempt, isFirstAttempt)
		if err := a.failedLoginsProvider.SaveFailedLoginAttempts(ctx, user.ID, newAttempt); err != nil {
			return "", fmt.Errorf("%s: %w", op, err)
		}

		return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	if err := a.failedLoginsProvider.RemoveFailedLoginAttempts(ctx, user.ID); err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	app, err := a.appProvider.App(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	token, err := jwt.NewToken(&user, app, a.tokenTTL)
	if err != nil {
		a.log.Error("failed to generate token", sl.Err(err))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return token, nil
}

func (a *Auth) RegisterNewUser(ctx context.Context, email string, password string) (uuid.UUID, error) {
	const op = "auth.RegisterNewUser"
	log := a.log.With(
		slog.String("op", op),
		slog.String("email", email),
	)
	log.Info("registering new user")

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate passwordHash", sl.Err(err))
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	user, err := a.userSaver.SaveUser(ctx, email, passwordHash)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			log.Error("user exists", sl.Err(err))
			return uuid.Nil, fmt.Errorf("%s: %w", op, ErrUserExists)
		}

		log.Error("failed to save user", sl.Err(err))
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user registered")

	return user.ID, nil
}

func (a *Auth) IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	const op = "auth.IsAdmin"
	log := a.log.With("op", op)
	log.Info("checking if user is admin")

	isAdmin, err := a.userProvider.IsAdmin(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Error("user not found", sl.Err(err))
			return false, fmt.Errorf("%s: %w", op, ErrInvalidUserID)
		}

		log.Error("failed to get IsAdmin", sl.Err(err))
		return false, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("checked if user is admin", slog.Bool("is_admin", isAdmin))

	return isAdmin, nil
}

func (a *Auth) CreateApp(ctx context.Context, name, secret string) (uuid.UUID, error) {
	const op = "auth.CreateApp"
	log := a.log.With("op", op)
	log.Info("creating new app")

	app, err := a.appProvider.CreateApp(ctx, name, secret)
	if err != nil {
		if errors.Is(err, storage.ErrAppExists) {
			log.Error("app exists", sl.Err(err))
			return uuid.Nil, fmt.Errorf("%s: %w", op, ErrAppExists)
		}

		log.Error("failed to create app", sl.Err(err))
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("created new app", slog.String("appName", app.Name))

	return app.ID, nil
}

func (a *Auth) App(ctx context.Context, name string) (models.App, error) {
	const op = "auth.App"
	log := a.log.With("op", op)
	log.Info("getting app")

	app, err := a.appProvider.FindApp(ctx, name)
	if err != nil {
		if errors.Is(err, storage.ErrAppNotFound) {
			log.Error("app not found", sl.Err(err))
			return models.App{}, fmt.Errorf("%s: %w", op, ErrAppNotFound)
		}

		log.Error("failed to get app", sl.Err(err))
		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	return app, nil
}

func (a *Auth) handleFailedLogin(userID uuid.UUID, failedLoginAttempt models.FailedLogin, isFirstAttempt bool) models.FailedLogin {
	now := time.Now()

	if isFirstAttempt {
		// First failed attempt, create a new tracker
		return models.FailedLogin{
			Attempts:  1,
			FirstFail: now,
		}
	}

	// Check if attempts are within the time window
	if now.Sub(failedLoginAttempt.FirstFail) > attemptWindow {
		// Reset tracker if the first attempt is outside the window
		return models.FailedLogin{
			Attempts:  1,
			FirstFail: now,
		}
	}

	// Increment the number of failed attempts
	failedLoginAttempt.Attempts++

	// Lock account if the threshold is exceeded
	if failedLoginAttempt.Attempts >= MaxFailedLoginAttempts {
		// Exponential Backoff
		newLockoutDuration := time.Duration(math.Pow(2, float64(failedLoginAttempt.Attempts-MaxFailedLoginAttempts))) * BaseLockoutDuration
		failedLoginAttempt.LockedUntil = now.Add(newLockoutDuration)
		a.log.Warn("account locked", slog.String("userID", userID.String()), slog.Time("lockedUntil", failedLoginAttempt.LockedUntil))
	}

	return failedLoginAttempt
}
