package post_service

import (
	"context"
	"pinstack-post-service/internal/model"
)

type Service interface {
	CreatePost(ctx context.Context, post *model.CreatePostDTO) (*model.Post, error)
	GetPostByID(ctx context.Context, id int) (*model.PostDetailed, error)
	ListPosts(ctx context.Context, filters *model.PostFilters) ([]*model.PostDetailed, error)
	UpdatePost(ctx context.Context, id int, post *model.UpdatePostDTO) error
	DeletePost(ctx context.Context, id int) error
}
