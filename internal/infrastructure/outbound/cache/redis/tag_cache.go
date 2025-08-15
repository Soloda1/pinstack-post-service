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
	tagsByPostKeyPrefix = "tags_by_post:"
	tagByNameKeyPrefix  = "tag_by_name:"
	tagCacheTTL         = 60 * time.Minute
)

type TagCache struct {
	client *Client
	log    ports.Logger
}

func NewTagCache(client *Client, log ports.Logger) *TagCache {
	return &TagCache{
		client: client,
		log:    log,
	}
}

func (t *TagCache) GetTagsByPost(ctx context.Context, postID int64) ([]*model.Tag, error) {
	key := t.getTagsByPostKey(postID)

	var tags []*model.Tag
	err := t.client.Get(ctx, key, &tags)
	if err != nil {
		if errors.Is(err, custom_errors.ErrCacheMiss) {
			t.log.Debug("Tags by post cache miss", slog.Int64("post_id", postID))
			return nil, custom_errors.ErrCacheMiss
		}
		t.log.Error("Failed to get tags by post from cache",
			slog.Int64("post_id", postID),
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to get tags by post from cache: %w", err)
	}

	t.log.Debug("Tags by post cache hit",
		slog.Int64("post_id", postID),
		slog.Int("tags_count", len(tags)))
	return tags, nil
}

func (t *TagCache) SetTagsByPost(ctx context.Context, postID int64, tags []*model.Tag) error {
	if tags == nil {
		tags = []*model.Tag{}
	}

	key := t.getTagsByPostKey(postID)

	if err := t.client.Set(ctx, key, tags, tagCacheTTL); err != nil {
		t.log.Error("Failed to set tags by post cache",
			slog.Int64("post_id", postID),
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to set tags by post cache: %w", err)
	}

	t.log.Debug("Tags by post cached successfully",
		slog.Int64("post_id", postID),
		slog.Int("tags_count", len(tags)),
		slog.Duration("ttl", tagCacheTTL))
	return nil
}

func (t *TagCache) DeleteTagsByPost(ctx context.Context, postID int64) error {
	key := t.getTagsByPostKey(postID)

	if err := t.client.Delete(ctx, key); err != nil {
		t.log.Error("Failed to delete tags by post from cache",
			slog.Int64("post_id", postID),
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to delete tags by post from cache: %w", err)
	}

	t.log.Debug("Tags by post deleted from cache", slog.Int64("post_id", postID))
	return nil
}

func (t *TagCache) GetTag(ctx context.Context, name string) (*model.Tag, error) {
	if name == "" {
		return nil, fmt.Errorf("tag name cannot be empty")
	}

	key := t.getTagByNameKey(name)

	var tag model.Tag
	err := t.client.Get(ctx, key, &tag)
	if err != nil {
		if err == custom_errors.ErrCacheMiss {
			t.log.Debug("Tag by name cache miss", slog.String("tag_name", name))
			return nil, custom_errors.ErrCacheMiss
		}
		t.log.Error("Failed to get tag by name from cache",
			slog.String("tag_name", name),
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to get tag by name from cache: %w", err)
	}

	t.log.Debug("Tag by name cache hit", slog.String("tag_name", name))
	return &tag, nil
}

func (t *TagCache) SetTag(ctx context.Context, tag *model.Tag) error {
	if tag == nil {
		return fmt.Errorf("tag cannot be nil")
	}
	if tag.Name == "" {
		return fmt.Errorf("tag name cannot be empty")
	}

	key := t.getTagByNameKey(tag.Name)

	if err := t.client.Set(ctx, key, tag, tagCacheTTL); err != nil {
		t.log.Error("Failed to set tag cache",
			slog.String("tag_name", tag.Name),
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to set tag cache: %w", err)
	}

	t.log.Debug("Tag cached successfully",
		slog.String("tag_name", tag.Name),
		slog.Int64("tag_id", tag.ID),
		slog.Duration("ttl", tagCacheTTL))
	return nil
}

func (t *TagCache) DeleteTag(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("tag name cannot be empty")
	}

	key := t.getTagByNameKey(name)

	if err := t.client.Delete(ctx, key); err != nil {
		t.log.Error("Failed to delete tag from cache",
			slog.String("tag_name", name),
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to delete tag from cache: %w", err)
	}

	t.log.Debug("Tag deleted from cache", slog.String("tag_name", name))
	return nil
}

func (t *TagCache) getTagsByPostKey(postID int64) string {
	return tagsByPostKeyPrefix + strconv.FormatInt(postID, 10)
}

func (t *TagCache) getTagByNameKey(name string) string {
	return tagByNameKeyPrefix + name
}
