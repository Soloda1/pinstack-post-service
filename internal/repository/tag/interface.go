package tag_repository

import (
	"context"
	"pinstack-post-service/internal/model"
)

//go:generate mockery --name Repository --dir . --output ../../../mocks --outpkg mocks --with-expecter --filename TagRepository.go
type Repository interface {
	FindByNames(ctx context.Context, names []string) ([]*model.Tag, error)
	FindByPost(ctx context.Context, postID int64) ([]*model.Tag, error)
	Create(ctx context.Context, name string) (*model.Tag, error)
	DeleteUnused(ctx context.Context) error
	TagPost(ctx context.Context, postID int64, tagNames []string) error
	UntagPost(ctx context.Context, postID int64, tagNames []string) error
	ReplacePostTags(ctx context.Context, postID int64, newTags []string) error
}
