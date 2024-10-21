package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/BariVakhidov/sso/internal/domain/converter"
	"github.com/BariVakhidov/sso/internal/domain/models"
	"github.com/BariVakhidov/sso/internal/lib/logger/sl"
	"github.com/BariVakhidov/sso/internal/storage"
	storageModel "github.com/BariVakhidov/sso/internal/storage/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	log    *slog.Logger
	dbpool *pgxpool.Pool
}

var (
	pgOnce sync.Once
)

func New(log *slog.Logger, dbAddr string) (*Storage, error) {
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

	return &Storage{dbpool: dbpool, log: log}, nil
}

func (s *Storage) SaveUser(ctx context.Context, userID, email string, passHash []byte) (user models.User, err error) {
	const op = "storage.postgres.SaveUser"
	log := s.log.With(slog.String("op", op))

	//TODO: TxOptions
	tx, err := s.dbpool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return user, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			rErr := tx.Rollback(ctx)
			if rErr != nil {
				log.Error("rollback failed", sl.Err(rErr))
			}
			return
		}

		if commitErr := tx.Commit(ctx); commitErr != nil {
			log.Error("commit failed", sl.Err(commitErr))
			err = fmt.Errorf("%s: %w", op, commitErr)
		}
	}()

	query := "INSERT INTO users(id,email,pass_hash) VALUES(@userId,@userEmail,@userPassHash) RETURNING id,email,pass_hash"
	args := pgx.NamedArgs{
		"userId":       userID,
		"userEmail":    email,
		"userPassHash": passHash,
	}

	storageUser := storageModel.User{}

	err = tx.QueryRow(
		ctx,
		query,
		args,
	).Scan(&storageUser.ID, &storageUser.Email, &storageUser.PassHash)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return user, fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}

		return user, fmt.Errorf("%s: %w", op, err)
	}

	eventPayload, err := json.Marshal(converter.ToUserEventFromStorage(storageUser))
	if err != nil {
		return user, fmt.Errorf("%s: %w", op, err)
	}

	if err = s.saveEvent(ctx, tx, storage.EventUserCreated, string(eventPayload)); err != nil {
		return user, fmt.Errorf("%s: %w", op, err)
	}

	return converter.ToUserFromStorage(storageUser), nil
}

func (s *Storage) User(ctx context.Context, email string) (models.User, error) {
	const op = "storage.postgres.User"

	query := "SELECT id,email,pass_hash FROM users WHERE email=$1"
	var user storageModel.User

	err := s.dbpool.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, &user.PassHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return converter.ToUserFromStorage(user), nil
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

func (s *Storage) NewEvents(ctx context.Context, limit int) ([]models.Event, error) {
	const op = "storage.postgres.NewEvents"

	query := `WITH selected_events AS (
			SELECT id, event_type, payload, status, created_at, reserved_to
			FROM events
			WHERE status = 'new'
			  AND (reserved_to IS NULL OR reserved_to < NOW())
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE events
		SET reserved_to = NOW() + INTERVAL '1 minute'
		FROM selected_events
		WHERE events.id = selected_events.id
		RETURNING selected_events.id, selected_events.event_type, selected_events.payload,selected_events.status,selected_events.created_at,selected_events.reserved_to;
	`

	rows, err := s.dbpool.Query(ctx, query, limit)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrEventsNotFound)
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	defer rows.Close()
	events, err := pgx.CollectRows(rows, pgx.RowToStructByName[storageModel.Event])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return converter.ToEventsFromStorage(events), nil
}

func (s *Storage) SetEventDone(ctx context.Context, eventId uuid.UUID) (models.Event, error) {
	const op = "storage.postgres.SetEventDone"

	query := "UPDATE events SET status='done',reserved_to=Null WHERE id=$1 RETURNING id,event_type,payload"

	event := storageModel.Event{}
	err := s.dbpool.QueryRow(ctx, query, eventId).Scan(&event.ID, &event.Type, &event.Payload)
	if err != nil {
		return models.Event{}, fmt.Errorf("%s: %w", op, err)
	}

	return converter.ToEventFromStorage(event), nil
}

func (s *Storage) saveEvent(ctx context.Context, tx pgx.Tx, eventType, payload string) error {
	const op = "storage.postgres.saveEvent"

	query := "INSERT INTO events(id,event_type,payload) VALUES(@eventId,@eventType,@payload)"
	args := pgx.NamedArgs{
		"eventId":   uuid.New(),
		"eventType": eventType,
		"payload":   payload,
	}

	if _, err := tx.Exec(
		ctx,
		query,
		args,
	); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) ClosePool() {
	s.dbpool.Close()
}
