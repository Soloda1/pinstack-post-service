package memory

import (
	"context"
	"log/slog"
	ports "pinstack-post-service/internal/domain/ports/output"
	"sort"
	"sync"
	"time"

	"github.com/soloda1/pinstack-proto-definitions/custom_errors"

	"github.com/jackc/pgx/v5/pgtype"
	model "pinstack-post-service/internal/domain/models"
)

type PostRepository struct {
	log    ports.Logger
	mu     sync.RWMutex
	posts  map[int64]*model.Post
	nextID int64
}

func NewPostRepository(log ports.Logger) *PostRepository {
	return &PostRepository{
		log:    log,
		posts:  make(map[int64]*model.Post),
		nextID: 1,
	}
}

func (p *PostRepository) Create(ctx context.Context, post *model.Post) (*model.Post, error) {
	p.log.Debug("Creating new post (memory impl)", slog.Int64("author_id", post.AuthorID), slog.String("title", post.Title))

	p.mu.Lock()
	defer p.mu.Unlock()

	now := pgtype.Timestamp{Time: time.Now(), Valid: true}

	newPost := &model.Post{
		ID:        p.nextID,
		AuthorID:  post.AuthorID,
		Title:     post.Title,
		Content:   post.Content,
		CreatedAt: now,
		UpdatedAt: now,
	}
	p.nextID++

	p.posts[newPost.ID] = newPost

	p.log.Debug("Successfully created post (memory impl)", slog.Int64("id", newPost.ID), slog.Int64("author_id", newPost.AuthorID))
	result := *newPost
	return &result, nil
}

func (p *PostRepository) GetByID(ctx context.Context, id int64) (*model.Post, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	post, exists := p.posts[id]
	if !exists {
		p.log.Debug("Post not found by id", slog.Int64("id", id))
		return nil, custom_errors.ErrPostNotFound
	}

	result := *post
	return &result, nil
}

func (p *PostRepository) GetByAuthor(ctx context.Context, authorID int64) ([]*model.Post, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result []*model.Post
	for _, post := range p.posts {
		if post.AuthorID == authorID {
			postCopy := *post
			result = append(result, &postCopy)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Time.After(result[j].CreatedAt.Time)
	})

	return result, nil
}

func (p *PostRepository) Update(ctx context.Context, id int64, update *model.UpdatePostDTO) (*model.Post, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	post, exists := p.posts[id]
	if !exists {
		return nil, custom_errors.ErrPostNotFound
	}

	if update.Title != nil {
		post.Title = *update.Title
	}
	if update.Content != nil {
		post.Content = update.Content
	}

	post.UpdatedAt = pgtype.Timestamp{Time: time.Now(), Valid: true}

	result := *post
	return &result, nil
}

func (p *PostRepository) Delete(ctx context.Context, id int64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.posts[id]; !exists {
		return custom_errors.ErrPostNotFound
	}

	delete(p.posts, id)
	return nil
}

func (p *PostRepository) List(ctx context.Context, filters model.PostFilters) ([]*model.Post, int, error) {
	p.log.Debug("Listing posts with filters (memory impl)",
		slog.Any("author_id", filters.AuthorID),
		slog.Any("created_after", filters.CreatedAfter),
		slog.Any("created_before", filters.CreatedBefore),
		slog.Any("tag_names", filters.TagNames),
		slog.Any("limit", filters.Limit),
		slog.Any("offset", filters.Offset))

	p.mu.RLock()
	defer p.mu.RUnlock()

	var filteredPosts []*model.Post
	for _, post := range p.posts {
		if filters.AuthorID != nil && post.AuthorID != *filters.AuthorID {
			p.log.Debug("Skipping post: author ID doesn't match", slog.Int64("post_id", post.ID),
				slog.Int64("post_author", post.AuthorID), slog.Int64("filter_author", *filters.AuthorID))
			continue
		}
		if filters.CreatedAfter != nil && (post.CreatedAt.Time.Before(filters.CreatedAfter.Time) || post.CreatedAt.Time.Equal(filters.CreatedAfter.Time)) {
			p.log.Debug("Skipping post: creation time not after filter", slog.Int64("post_id", post.ID),
				slog.Time("post_time", post.CreatedAt.Time), slog.Time("filter_time", filters.CreatedAfter.Time))
			continue
		}
		if filters.CreatedBefore != nil && post.CreatedAt.Time.After(filters.CreatedBefore.Time) {
			p.log.Debug("Skipping post: creation time after filter", slog.Int64("post_id", post.ID),
				slog.Time("post_time", post.CreatedAt.Time), slog.Time("filter_time", filters.CreatedBefore.Time))
			continue
		}
		// TagNames filtering not implemented in memory repository

		p.log.Debug("Post passed all filters", slog.Int64("post_id", post.ID))
		postCopy := *post
		filteredPosts = append(filteredPosts, &postCopy)
	}

	sort.Slice(filteredPosts, func(i, j int) bool {
		return filteredPosts[i].CreatedAt.Time.After(filteredPosts[j].CreatedAt.Time)
	})

	total := len(filteredPosts)
	p.log.Debug("Total matching posts before pagination", slog.Int("total", total))

	// Apply offset
	if filters.Offset != nil {
		offset := int(*filters.Offset)
		p.log.Debug("Applying offset", slog.Int("offset", offset))
		if offset >= len(filteredPosts) {
			p.log.Debug("Offset exceeds results count, returning empty list",
				slog.Int("offset", offset), slog.Int("results_count", len(filteredPosts)))
			return []*model.Post{}, total, nil
		}
		filteredPosts = filteredPosts[offset:]
	}

	// Apply limit
	if filters.Limit != nil {
		limit := int(*filters.Limit)
		p.log.Debug("Applying limit", slog.Int("limit", limit), slog.Int("results_count", len(filteredPosts)))
		if limit < len(filteredPosts) {
			filteredPosts = filteredPosts[:limit]
		}
	}

	p.log.Debug("Returning filtered posts", slog.Int("count", len(filteredPosts)), slog.Int("total", total))
	return filteredPosts, total, nil
}
