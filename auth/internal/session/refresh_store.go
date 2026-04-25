package session

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RefreshStore interface {
	Save(ctx context.Context, tokenID string, userID int64, ttl time.Duration) error
	ExistsForUser(ctx context.Context, tokenID string, userID int64) (bool, error)
	Revoke(ctx context.Context, tokenID string) error
}

type redisRefreshStore struct {
	client *redis.Client
}

func NewRedisRefreshStore(client *redis.Client) RefreshStore {
	return &redisRefreshStore{client: client}
}

func (s *redisRefreshStore) Save(ctx context.Context, tokenID string, userID int64, ttl time.Duration) error {
	return s.client.Set(ctx, refreshKey(tokenID), strconv.FormatInt(userID, 10), ttl).Err()
}

func (s *redisRefreshStore) ExistsForUser(ctx context.Context, tokenID string, userID int64) (bool, error) {
	storedUserID, err := s.client.Get(ctx, refreshKey(tokenID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}

	return storedUserID == strconv.FormatInt(userID, 10), nil
}

func (s *redisRefreshStore) Revoke(ctx context.Context, tokenID string) error {
	return s.client.Del(ctx, refreshKey(tokenID)).Err()
}

func refreshKey(tokenID string) string {
	return "auth:refresh:" + tokenID
}
