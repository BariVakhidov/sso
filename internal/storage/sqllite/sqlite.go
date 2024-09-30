package sqllite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/BariVakhidov/sso/internal/domain/models"
	"github.com/BariVakhidov/sso/internal/storage"
	"github.com/google/uuid"
	"github.com/mattn/go-sqlite3"
)

type Storage struct {
	db *sql.DB
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) SaveUser(ctx context.Context, email string, passHash []byte) (models.User, error) {
	const op = "storage.sqlite.SaveUser"

	stmt, err := s.db.Prepare("INSERT INTO users(id,email,pass_hash) VALUES(?,?,?) RETURNING id,email,pass_hash")
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	user := models.User{}

	row := stmt.QueryRowContext(ctx, uuid.New(), email, passHash)
	if err := row.Scan(&user.ID, &user.Email, &user.PassHash); err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}

		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

func (s *Storage) User(ctx context.Context, email string) (models.User, error) {
	const op = "storage.sqlite.User"

	stmt, err := s.db.Prepare("SELECT id,email,pass_hash FROM users WHERE email=?")
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	row := stmt.QueryRowContext(ctx, email)

	var user models.User
	if err := row.Scan(&user.ID, &user.Email, &user.PassHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

func (s *Storage) IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	const op = "storage.sqlite.IsAdmin"
	stmt, err := s.db.Prepare("SELECT is_admin FROM users WHERE id=?")
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	res := stmt.QueryRowContext(ctx, userID)
	var isAdmin bool
	if err := res.Scan(&isAdmin); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}

		return false, fmt.Errorf("%s: %w", op, err)
	}

	return isAdmin, nil
}

func (s *Storage) App(ctx context.Context, appID uuid.UUID) (models.App, error) {
	const op = "storage.sqlite.App"

	stmt, err := s.db.Prepare("SELECT id,name,secret FROM apps WHERE id=?")
	if err != nil {
		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	res := stmt.QueryRowContext(ctx, appID)

	var app models.App
	if err := res.Scan(&app.ID, &app.Name, &app.Secret); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.App{}, fmt.Errorf("%s: %w", op, storage.ErrAppNotFound)
		}

		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	return app, nil
}

func (s *Storage) FindApp(ctx context.Context, name string) (models.App, error) {
	const op = "storage.sqlite.FindApp"

	stmt, err := s.db.Prepare("SELECT id,name,secret FROM apps WHERE name=?")
	if err != nil {
		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	res := stmt.QueryRowContext(ctx, name)

	var app models.App
	if err := res.Scan(&app.ID, &app.Name, &app.Secret); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.App{}, fmt.Errorf("%s: %w", op, storage.ErrAppNotFound)
		}

		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	return app, nil
}

func (s *Storage) CreateApp(ctx context.Context, name string, secret string) (models.App, error) {
	const op = "storage.sqlite.CreateApp"

	stmt, err := s.db.Prepare("INSERT INTO apps(id,name,secret) VALUES(?,?,?) RETURNING id,name,secret")
	if err != nil {
		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	res := stmt.QueryRowContext(ctx, uuid.New(), name, secret)

	var app models.App
	if err := res.Scan(&app.ID, &app.Name, &app.Secret); err != nil {
		fmt.Println(err)
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return models.App{}, fmt.Errorf("%s: %w", op, storage.ErrAppExists)
		}

		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	return app, nil
}
