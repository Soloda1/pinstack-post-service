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
	postCacheKeyPrefix = "post:"
	postCacheTTL       = 30 * time.Minute
)

type PostCache struct {
	client *Client
	log    ports.Logger
}

func NewPostCache(client *Client, log ports.Logger) *PostCache {
	return &PostCache{
		client: client,
		log:    log,
	}
}

func (p *PostCache) GetPost(ctx context.Context, postID int64) (*model.PostDetailed, error) {
	key := p.getPostKey(postID)

	var post model.PostDetailed
	err := p.client.Get(ctx, key, &post)
	if err != nil {
		if errors.Is(err, custom_errors.ErrCacheMiss) {
			p.log.Debug("Post cache miss", slog.Int64("post_id", postID))
			return nil, custom_errors.ErrCacheMiss
		}
		p.log.Error("Failed to get post from cache",
			slog.Int64("post_id", postID),
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to get post from cache: %w", err)
	}

	p.log.Debug("Post cache hit", slog.Int64("post_id", postID))
	return &post, nil
}

func (p *PostCache) SetPost(ctx context.Context, post *model.PostDetailed) error {
	if post == nil {
		return fmt.Errorf("post cannot be nil")
	}
	if post.Post == nil {
		return fmt.Errorf("post.Post cannot be nil")
	}

	key := p.getPostKey(post.Post.ID)

	if err := p.client.Set(ctx, key, post, postCacheTTL); err != nil {
		p.log.Error("Failed to set post cache",
			slog.Int64("post_id", post.Post.ID),
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to set post cache: %w", err)
	}

	p.log.Debug("Post cached successfully",
		slog.Int64("post_id", post.Post.ID),
		slog.Duration("ttl", postCacheTTL))
	return nil
}

func (p *PostCache) DeletePost(ctx context.Context, postID int64) error {
	key := p.getPostKey(postID)

	if err := p.client.Delete(ctx, key); err != nil {
		p.log.Error("Failed to delete post from cache",
			slog.Int64("post_id", postID),
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to delete post from cache: %w", err)
	}

	p.log.Debug("Post deleted from cache", slog.Int64("post_id", postID))
	return nil
}

func (p *PostCache) getPostKey(postID int64) string {
	return postCacheKeyPrefix + strconv.FormatInt(postID, 10)
}
