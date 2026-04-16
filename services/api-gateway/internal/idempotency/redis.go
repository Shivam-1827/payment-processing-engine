package idempotency

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	StatusProcessing = "PROCESSING"
	KeyExpiration    = 24 * time.Hour
)

var ErrConcurrentRequest = errors.New("request is already processing")

type Store interface {
	AcquireLock(ctx context.Context, key string) (isNew bool, cachedVal string, err error)
	SaveResponse(ctx context.Context, key string, responseJSON string) error
	Delete(ctx context.Context, key string) error
}

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

func (s *RedisStore) AcquireLock(ctx context.Context, key string) (bool, string, error) {
	locked, err := s.client.SetNX(ctx, key, StatusProcessing, KeyExpiration).Result()
	if err != nil {
		return false, "", fmt.Errorf("redis SetNX failed: %w", err)
	}

	if locked {
		return true, "", nil
	}

	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return false, "", fmt.Errorf("redis Get failed: %w", err)
	}

	if val == StatusProcessing {
		return false, "", ErrConcurrentRequest
	}

	return false, val, nil
}

func (s *RedisStore) SaveResponse(ctx context.Context, key string, responseJSON string) error {
	return s.client.Set(ctx, key, responseJSON, KeyExpiration).Err()
}

func (s *RedisStore) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}
