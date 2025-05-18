package post_repository

import (
	"context"
	"pinstack-post-service/internal/model"
)

type Repository interface {
	Create(ctx context.Context, post *model.Post) (*model.Post, error)
	GetByID(ctx context.Context, id int64) (*model.Post, error)
	GetByAuthor(ctx context.Context, authorID int64) ([]*model.Post, error)
	UpdateTitle(ctx context.Context, id int64, title string) (*model.Post, error)
	UpdateContent(ctx context.Context, id int64, content string) (*model.Post, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, filters model.PostFilters) ([]*model.Post, error)
}
