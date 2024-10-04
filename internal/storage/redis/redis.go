package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/BariVakhidov/sso/internal/domain/models"
	"github.com/BariVakhidov/sso/internal/storage"
	"github.com/redis/go-redis/v9"
)

type Storage struct {
	client *redis.Client
	ttl    time.Duration
}

func New(addr string, ttl time.Duration) *Storage {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	return &Storage{client: client, ttl: ttl}
}

func (s *Storage) FailedLoginAttempts(ctx context.Context, userId string) (models.FailedLogin, error) {
	const op = "storage.redis.FailedLoginAttempts"

	data := s.client.Get(ctx, fmt.Sprintf("failedLogin:%s", userId)).Val()

	if len(data) == 0 {
		return models.FailedLogin{}, fmt.Errorf("%s: %w", op, storage.ErrFailedLoginNotFound)
	}

	var failedLoginAttempt models.FailedLogin
	err := json.Unmarshal([]byte(data), &failedLoginAttempt)
	if err != nil {
		return models.FailedLogin{}, fmt.Errorf("%s: %w", op, err)
	}

	return failedLoginAttempt, nil
}

func (s *Storage) RemoveFailedLoginAttempts(ctx context.Context, userId string) error {
	const op = "storage.redis.RemoveFailedLoginAttempts"

	if err := s.client.Del(ctx, fmt.Sprintf("failedLogin:%s", userId)).Err(); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) SaveFailedLoginAttempts(ctx context.Context, userId string, failedLoginAttempt models.FailedLogin) error {
	const op = "storage.redis.SaveFailedLoginAttempts"

	data, err := json.Marshal(failedLoginAttempt)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if err = s.client.Set(ctx, fmt.Sprintf("failedLogin:%s", userId), string(data), s.ttl).Err(); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) Stop() error {
	const op = "storage.redis.Stop"

	if err := s.client.Close(); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
