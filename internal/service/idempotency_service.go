package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type IdempotencyService struct {
	redis *redis.Client
	ttl   time.Duration
}

func NewIdempotencyService(redis *redis.Client, ttl time.Duration) *IdempotencyService {
	return &IdempotencyService{redis: redis, ttl: ttl}
}

func (s *IdempotencyService) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if key == "" {
		return nil, false, nil
	}
	value, err := s.redis.Get(ctx, "idem:"+key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("idempotency_service.Get: %w", err)
	}
	return value, true, nil
}

func (s *IdempotencyService) Store(ctx context.Context, key string, response []byte) error {
	if key == "" {
		return nil
	}
	if err := s.redis.Set(ctx, "idem:"+key, response, s.ttl).Err(); err != nil {
		return fmt.Errorf("idempotency_service.Store: %w", err)
	}
	return nil
}
