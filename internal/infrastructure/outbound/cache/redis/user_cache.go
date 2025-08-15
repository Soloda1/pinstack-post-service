package redis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"pinstack-post-service/internal/domain/models"
	ports "pinstack-post-service/internal/domain/ports/output"

	"github.com/soloda1/pinstack-proto-definitions/custom_errors"
)

const (
	userCacheKeyPrefix = "user:"
	userCacheTTL       = 15 * time.Minute
)

type UserCache struct {
	client *Client
	log    ports.Logger
}

func NewUserCache(client *Client, log ports.Logger) *UserCache {
	return &UserCache{
		client: client,
		log:    log,
	}
}

func (u *UserCache) GetUser(ctx context.Context, userID int64) (*model.User, error) {
	key := u.getUserKey(userID)

	var user model.User
	err := u.client.Get(ctx, key, &user)
	if err != nil {
		if errors.Is(err, custom_errors.ErrCacheMiss) {
			u.log.Debug("User cache miss", slog.Int64("user_id", userID))
			return nil, custom_errors.ErrCacheMiss
		}
		u.log.Error("Failed to get user from cache",
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to get user from cache: %w", err)
	}

	u.log.Debug("User cache hit", slog.Int64("user_id", userID))
	return &user, nil
}

func (u *UserCache) SetUser(ctx context.Context, user *model.User) error {
	if user == nil {
		return fmt.Errorf("user cannot be nil")
	}

	key := u.getUserKey(user.ID)

	if err := u.client.Set(ctx, key, user, userCacheTTL); err != nil {
		u.log.Error("Failed to set user cache",
			slog.Int64("user_id", user.ID),
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to set user cache: %w", err)
	}

	u.log.Debug("User cached successfully",
		slog.Int64("user_id", user.ID),
		slog.Duration("ttl", userCacheTTL))
	return nil
}

func (u *UserCache) DeleteUser(ctx context.Context, userID int64) error {
	key := u.getUserKey(userID)

	if err := u.client.Delete(ctx, key); err != nil {
		u.log.Error("Failed to delete user from cache",
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to delete user from cache: %w", err)
	}

	u.log.Debug("User deleted from cache", slog.Int64("user_id", userID))
	return nil
}

func (u *UserCache) getUserKey(userID int64) string {
	return userCacheKeyPrefix + strconv.FormatInt(userID, 10)
}
