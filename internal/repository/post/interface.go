package post_repository

import (
	"context"
	"pinstack-post-service/internal/model"
)

type Repository interface {
	Create(ctx context.Context, post *model.Post) error
	GetByID(ctx context.Context, id int) (*model.Post, error)
	GetByAuthor(ctx context.Context, authorID int64) ([]*model.Post, error)
	UpdateTitle(ctx context.Context, id int, title string) error
	UpdateContent(ctx context.Context, id int, content string) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, filters model.PostFilters) ([]*model.Post, error)
}
