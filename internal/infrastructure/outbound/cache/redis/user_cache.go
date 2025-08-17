package redis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	model "pinstack-post-service/internal/domain/models"
	ports "pinstack-post-service/internal/domain/ports/output"

	"github.com/soloda1/pinstack-proto-definitions/custom_errors"
)

const (
	userCacheKeyPrefix = "user:"
	userCacheTTL       = 15 * time.Minute
)

type UserCache struct {
	client  *Client
	log     ports.Logger
	metrics ports.MetricsProvider
}

func NewUserCache(client *Client, log ports.Logger, metrics ports.MetricsProvider) *UserCache {
	return &UserCache{
		client:  client,
		log:     log,
		metrics: metrics,
	}
}

func (u *UserCache) GetUser(ctx context.Context, userID int64) (*model.User, error) {
	start := time.Now()
	key := u.getUserKey(userID)

	var user model.User
	err := u.client.Get(ctx, key, &user)
	if err != nil {
		if errors.Is(err, custom_errors.ErrCacheMiss) {
			u.log.Debug("User cache miss", slog.Int64("user_id", userID))
			u.metrics.IncrementCacheMisses()
			u.metrics.RecordCacheMissDuration("user_get", time.Since(start))
			return nil, custom_errors.ErrCacheMiss
		}
		u.log.Error("Failed to get user from cache",
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()))
		u.metrics.RecordCacheOperationDuration("user_get", time.Since(start))
		return nil, fmt.Errorf("failed to get user from cache: %w", err)
	}

	u.metrics.IncrementCacheHits()
	u.metrics.RecordCacheHitDuration("user_get", time.Since(start))
	u.log.Debug("User cache hit", slog.Int64("user_id", userID))
	return &user, nil
}

func (u *UserCache) SetUser(ctx context.Context, user *model.User) error {
	start := time.Now()
	if user == nil {
		return fmt.Errorf("user cannot be nil")
	}

	key := u.getUserKey(user.ID)

	if err := u.client.Set(ctx, key, user, userCacheTTL); err != nil {
		u.log.Error("Failed to set user cache",
			slog.Int64("user_id", user.ID),
			slog.String("error", err.Error()))
		u.metrics.RecordCacheOperationDuration("user_set", time.Since(start))
		return fmt.Errorf("failed to set user cache: %w", err)
	}

	u.metrics.RecordCacheOperationDuration("user_set", time.Since(start))
	u.log.Debug("User cached successfully",
		slog.Int64("user_id", user.ID),
		slog.Duration("ttl", userCacheTTL))
	return nil
}

func (u *UserCache) DeleteUser(ctx context.Context, userID int64) error {
	start := time.Now()
	key := u.getUserKey(userID)

	if err := u.client.Delete(ctx, key); err != nil {
		u.log.Error("Failed to delete user from cache",
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()))
		u.metrics.RecordCacheOperationDuration("user_delete", time.Since(start))
		return fmt.Errorf("failed to delete user from cache: %w", err)
	}

	u.metrics.RecordCacheOperationDuration("user_delete", time.Since(start))
	u.log.Debug("User deleted from cache", slog.Int64("user_id", userID))
	return nil
}

func (u *UserCache) getUserKey(userID int64) string {
	return userCacheKeyPrefix + strconv.FormatInt(userID, 10)
}
