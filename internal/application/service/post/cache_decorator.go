package post_service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	model "pinstack-post-service/internal/domain/models"
	output "pinstack-post-service/internal/domain/ports/output"
	"pinstack-post-service/internal/domain/ports/output/cache"

	post_service "pinstack-post-service/internal/domain/ports/input/post"

	"github.com/soloda1/pinstack-proto-definitions/custom_errors"
)

type PostServiceCacheDecorator struct {
	service   post_service.Service
	userCache cache.UserCache
	postCache cache.PostCache
	log       output.Logger
	metrics   output.MetricsProvider
}

func NewPostServiceCacheDecorator(
	service post_service.Service,
	userCache cache.UserCache,
	postCache cache.PostCache,
	log output.Logger,
	metrics output.MetricsProvider,
) post_service.Service {
	return &PostServiceCacheDecorator{
		service:   service,
		userCache: userCache,
		postCache: postCache,
		log:       log,
		metrics:   metrics,
	}
}

func (d *PostServiceCacheDecorator) CreatePost(ctx context.Context, post *model.CreatePostDTO) (*model.PostDetailed, error) {
	d.log.Debug("Creating post with cache decorator", slog.Int64("author_id", post.AuthorID))

	result, err := d.service.CreatePost(ctx, post)
	if err != nil {
		return nil, err
	}

	if err := d.userCache.DeleteUser(ctx, post.AuthorID); err != nil {
		d.log.Warn("Failed to invalidate user cache after post creation",
			slog.Int64("user_id", post.AuthorID),
			slog.String("error", err.Error()))
	}

	start := time.Now()
	if err := d.postCache.SetPost(ctx, result); err != nil {
		d.log.Warn("Failed to cache created post",
			slog.Int64("post_id", result.Post.ID),
			slog.String("error", err.Error()))
		d.metrics.RecordCacheOperationDuration("post_set", time.Since(start))
	} else {
		d.metrics.RecordCacheOperationDuration("post_set", time.Since(start))
	}

	if result.Author != nil {
		userCacheStart := time.Now()
		if err := d.userCache.SetUser(ctx, result.Author); err != nil {
			d.log.Warn("Failed to cache author after post creation",
				slog.Int64("user_id", result.Author.ID),
				slog.String("error", err.Error()))
			d.metrics.RecordCacheOperationDuration("user_set", time.Since(userCacheStart))
		} else {
			d.metrics.RecordCacheOperationDuration("user_set", time.Since(userCacheStart))
		}
	}

	return result, nil
}

func (d *PostServiceCacheDecorator) GetPostByID(ctx context.Context, id int64) (*model.PostDetailed, error) {
	d.log.Debug("Getting post by ID with cache decorator", slog.Int64("post_id", id))

	cacheStart := time.Now()
	cachedPost, err := d.postCache.GetPost(ctx, id)
	d.metrics.RecordCacheOperationDuration("post_get", time.Since(cacheStart))
	if err == nil {
		d.log.Debug("Post found in cache", slog.Int64("post_id", id))
		d.metrics.IncrementCacheHits()
		return cachedPost, nil
	}

	if !errors.Is(err, custom_errors.ErrCacheMiss) {
		d.log.Warn("Failed to get post from cache",
			slog.Int64("post_id", id),
			slog.String("error", err.Error()))
	} else {
		d.metrics.IncrementCacheMisses()
	}

	d.log.Debug("Post cache miss, fetching from service", slog.Int64("post_id", id))
	post, err := d.service.GetPostByID(ctx, id)
	if err != nil {
		return nil, err
	}

	setCacheStart := time.Now()
	if err := d.postCache.SetPost(ctx, post); err != nil {
		d.log.Warn("Failed to cache post",
			slog.Int64("post_id", id),
			slog.String("error", err.Error()))
		d.metrics.RecordCacheOperationDuration("post_set", time.Since(setCacheStart))
	} else {
		d.metrics.RecordCacheOperationDuration("post_set", time.Since(setCacheStart))
	}

	if post.Author != nil {
		userCacheStart := time.Now()
		if err := d.userCache.SetUser(ctx, post.Author); err != nil {
			d.log.Warn("Failed to cache author",
				slog.Int64("user_id", post.Author.ID),
				slog.String("error", err.Error()))
			d.metrics.RecordCacheOperationDuration("user_set", time.Since(userCacheStart))
		} else {
			d.metrics.RecordCacheOperationDuration("user_set", time.Since(userCacheStart))
		}
	}

	return post, nil
}

func (d *PostServiceCacheDecorator) ListPosts(ctx context.Context, filters *model.PostFilters) ([]*model.PostDetailed, int, error) {
	d.log.Debug("Listing posts with cache decorator")

	posts, total, err := d.service.ListPosts(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	authorIDs := make(map[int64]bool)
	for _, post := range posts {
		if post.Post != nil {
			authorIDs[post.Post.AuthorID] = true
		}
	}

	for authorID := range authorIDs {
		userGetStart := time.Now()
		if cachedUser, err := d.userCache.GetUser(ctx, authorID); err == nil {
			d.log.Debug("Author found in cache", slog.Int64("author_id", authorID))
			d.metrics.RecordCacheOperationDuration("user_get", time.Since(userGetStart))
			for _, post := range posts {
				if post.Post != nil && post.Post.AuthorID == authorID {
					post.Author = cachedUser
				}
			}
		} else {
			d.metrics.RecordCacheOperationDuration("user_get", time.Since(userGetStart))
			for _, post := range posts {
				if post.Post != nil && post.Post.AuthorID == authorID && post.Author != nil {
					userSetStart := time.Now()
					if setErr := d.userCache.SetUser(ctx, post.Author); setErr != nil {
						d.log.Warn("Failed to cache author from list",
							slog.Int64("author_id", authorID),
							slog.String("error", setErr.Error()))
						d.metrics.RecordCacheOperationDuration("user_set", time.Since(userSetStart))
					} else {
						d.metrics.RecordCacheOperationDuration("user_set", time.Since(userSetStart))
					}
					break
				}
			}
		}
	}

	return posts, total, nil
}

func (d *PostServiceCacheDecorator) UpdatePost(ctx context.Context, userID int64, id int64, post *model.UpdatePostDTO) error {
	d.log.Debug("Updating post with cache decorator",
		slog.Int64("post_id", id),
		slog.Int64("user_id", userID))

	err := d.service.UpdatePost(ctx, userID, id, post)
	if err != nil {
		return err
	}

	cacheStart := time.Now()
	if err := d.postCache.DeletePost(ctx, id); err != nil {
		d.log.Warn("Failed to invalidate post cache after update",
			slog.Int64("post_id", id),
			slog.String("error", err.Error()))
		d.metrics.RecordCacheOperationDuration("post_delete", time.Since(cacheStart))
	} else {
		d.metrics.RecordCacheOperationDuration("post_delete", time.Since(cacheStart))
	}

	return nil
}

func (d *PostServiceCacheDecorator) DeletePost(ctx context.Context, userID int64, id int64) error {
	d.log.Debug("Deleting post with cache decorator",
		slog.Int64("post_id", id),
		slog.Int64("user_id", userID))

	err := d.service.DeletePost(ctx, userID, id)
	if err != nil {
		return err
	}

	cacheStart := time.Now()
	if err := d.postCache.DeletePost(ctx, id); err != nil {
		d.log.Warn("Failed to invalidate post cache after deletion",
			slog.Int64("post_id", id),
			slog.String("error", err.Error()))
		d.metrics.RecordCacheOperationDuration("post_delete", time.Since(cacheStart))
	} else {
		d.metrics.RecordCacheOperationDuration("post_delete", time.Since(cacheStart))
	}

	return nil
}
