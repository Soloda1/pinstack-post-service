package memory

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"pinstack-post-service/internal/custom_errors"
	"pinstack-post-service/internal/logger"
	"pinstack-post-service/internal/model"
)

type PostRepository struct {
	log    *logger.Logger
	mu     sync.RWMutex
	posts  map[int64]*model.Post
	nextID int64
}

func NewPostRepository(log *logger.Logger) *PostRepository {
	return &PostRepository{
		log:    log,
		posts:  make(map[int64]*model.Post),
		nextID: 1,
	}
}

func (p *PostRepository) Create(ctx context.Context, post *model.Post) (*model.Post, error) {
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

func (p *PostRepository) List(ctx context.Context, filters model.PostFilters) ([]*model.Post, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result []*model.Post
	for _, post := range p.posts {
		if filters.AuthorID != nil && post.AuthorID != *filters.AuthorID {
			continue
		}
		if filters.CreatedAfter != nil && post.CreatedAt.Time.Before(filters.CreatedAfter.Time) {
			continue
		}
		if filters.CreatedBefore != nil && post.CreatedAt.Time.After(filters.CreatedBefore.Time) {
			continue
		}

		postCopy := *post
		result = append(result, &postCopy)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Time.After(result[j].CreatedAt.Time)
	})

	if filters.Offset != nil {
		offset := int(*filters.Offset)
		if offset >= len(result) {
			return []*model.Post{}, nil
		}
		result = result[offset:]
	}

	if filters.Limit != nil {
		limit := int(*filters.Limit)
		if limit < len(result) {
			result = result[:limit]
		}
	}

	return result, nil
}
