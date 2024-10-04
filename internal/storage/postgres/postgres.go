package postgres

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/BariVakhidov/sso/internal/domain/models"
	"github.com/BariVakhidov/sso/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	dbpool *pgxpool.Pool
}

var (
	pgOnce sync.Once
)

func New(dbAddr string) (*Storage, error) {
	const op = "storage.postgres.New"

	var (
		dbpool *pgxpool.Pool
		err    error
	)

	//single instance of the db
	pgOnce.Do(func() {
		dbpool, err = pgxpool.New(context.Background(), dbAddr)
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{dbpool: dbpool}, nil
}

func (s *Storage) SaveUser(ctx context.Context, userID, email string, passHash []byte) (models.User, error) {
	const op = "storage.postgres.SaveUser"

	query := "INSERT INTO users(id,email,pass_hash) VALUES(@userId,@userEmail,@userPassHash) RETURNING id,email,pass_hash"
	args := pgx.NamedArgs{
		"userId":       userID,
		"userEmail":    email,
		"userPassHash": passHash,
	}

	user := models.User{}
	err := s.dbpool.QueryRow(
		ctx,
		query,
		args,
	).Scan(&user.ID, &user.Email, &user.PassHash)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}

		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

func (s *Storage) User(ctx context.Context, email string) (models.User, error) {
	const op = "storage.postgres.User"

	query := "SELECT id,email,pass_hash FROM users WHERE email=$1"
	var user models.User

	err := s.dbpool.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, &user.PassHash)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

func (s *Storage) IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	const op = "storage.postgres.IsAdmin"

	query := "SELECT is_admin FROM users WHERE id=$1"
	var isAdmin bool
	err := s.dbpool.QueryRow(ctx, query, userID).Scan(&isAdmin)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}

		return false, fmt.Errorf("%s: %w", op, err)
	}

	return isAdmin, nil
}

func (s *Storage) App(ctx context.Context, appID uuid.UUID) (models.App, error) {
	const op = "storage.postgres.App"

	query := "SELECT id,name,secret FROM apps WHERE id=$1"
	var app models.App
	err := s.dbpool.QueryRow(ctx, query, appID).Scan(&app.ID, &app.Name, &app.Secret)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.App{}, fmt.Errorf("%s: %w", op, storage.ErrAppNotFound)
		}

		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	return app, nil
}

func (s *Storage) FindApp(ctx context.Context, name string) (models.App, error) {
	const op = "storage.postgres.FindApp"
	var app models.App

	query := "SELECT id,name,secret FROM apps WHERE name=$1"
	err := s.dbpool.QueryRow(ctx, query, name).Scan(&app.ID, &app.Name, &app.Secret)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.App{}, fmt.Errorf("%s: %w", op, storage.ErrAppNotFound)
		}

		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	return app, nil
}

func (s *Storage) CreateApp(ctx context.Context, appID, name string, secret string) (models.App, error) {
	const op = "storage.postgres.CreateApp"

	query := "INSERT INTO apps(id,name,secret) VALUES(@appId,@appName,@appSecret) RETURNING id,name,secret"
	args := pgx.NamedArgs{
		"appId":     appID,
		"appName":   name,
		"appSecret": secret,
	}
	var app models.App
	err := s.dbpool.QueryRow(ctx, query, args).Scan(&app.ID, &app.Name, &app.Secret)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.App{}, fmt.Errorf("%s: %w", op, storage.ErrAppExists)
		}

		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	return app, nil
}

func (s *Storage) ClosePool() {
	s.dbpool.Close()
}
