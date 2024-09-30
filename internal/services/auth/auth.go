package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/BariVakhidov/sso/internal/domain/models"
	"github.com/BariVakhidov/sso/internal/lib/jwt"
	"github.com/BariVakhidov/sso/internal/storage"
	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidUserID      = errors.New("invalid userID")
	ErrUserExists         = errors.New("user exists")
	ErrAppExists          = errors.New("app exists")
	ErrAppNotFound        = errors.New("app not found")
	ErrUserNotFound       = errors.New("user not found")
)

type Auth struct {
	log          *slog.Logger
	userSaver    UserSaver
	userProvider UserProvider
	appProvider  AppProvider
	tokenTTL     time.Duration
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

// New returns a new instance of the Auth service
func New(
	log *slog.Logger,
	userSaver UserSaver,
	userProvider UserProvider,
	appProvider AppProvider,
	tokenTTL time.Duration,
) *Auth {
	return &Auth{
		log:          log,
		userSaver:    userSaver,
		userProvider: userProvider,
		appProvider:  appProvider,
		tokenTTL:     tokenTTL,
	}
}

func (a *Auth) Login(ctx context.Context, email string, password string, appID uuid.UUID) (string, error) {
	const op = "auth.Login"
	log := a.log.With("op", op)
	log.Info("login user")

	user, err := a.userProvider.User(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			a.log.Warn("user not found", "err", err)
			return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}

		log.Error("failed to get user", "err", err)
		return "", fmt.Errorf("%s: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		log.Error("invalid credentials", "err", err)
		return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	app, err := a.appProvider.App(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	token, err := jwt.NewToken(&user, app, a.tokenTTL)
	if err != nil {
		a.log.Error("failed to generate token", "err", err)
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return token, nil
}

func (a *Auth) RegisterNewUser(ctx context.Context, email string, password string) (uuid.UUID, error) {
	const op = "auth.RegisterNewUser"
	log := a.log.With("op", op)
	log.Info("registering new user")

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate passwordHash", "err", err)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	user, err := a.userSaver.SaveUser(ctx, email, passwordHash)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			log.Error("user exists", "err", err)
			return uuid.Nil, fmt.Errorf("%s: %w", op, ErrUserExists)
		}

		log.Error("failed to save user", "err", err)
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
			log.Error("user not found", "err", err)
			return false, fmt.Errorf("%s: %w", op, ErrInvalidUserID)
		}

		log.Error("failed to get IsAdmin", "err", err)
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
			log.Error("app exists", "err", err)
			return uuid.Nil, fmt.Errorf("%s: %w", op, ErrAppExists)
		}

		log.Error("failed to create app", "err", err)
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
			log.Error("app not found", "err", err)
			return models.App{}, fmt.Errorf("%s: %w", op, ErrAppNotFound)
		}

		log.Error("failed to get app", "err", err)
		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	return app, nil
}
