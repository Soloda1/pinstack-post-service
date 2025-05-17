package tag_repository

import (
	"context"
	"pinstack-post-service/internal/model"
)

type Repository interface {
	FindByNames(ctx context.Context, names []string) ([]*model.PostTag, error)
	FindByPost(ctx context.Context, postID int) ([]*model.PostTag, error)
	Create(ctx context.Context, name string) (*model.PostTag, error)
	DeleteUnused(ctx context.Context) error
	TagPost(ctx context.Context, postID int, tagNames []string) error
	UntagPost(ctx context.Context, postID int, tagNames []string) error
	ReplacePostTags(ctx context.Context, postID int, newTags []string) error
}
