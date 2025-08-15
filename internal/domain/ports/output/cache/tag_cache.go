package cache

import (
	"context"
	model "pinstack-post-service/internal/domain/models"
)

//go:generate mockery --name TagCache --dir . --output ../../../../mocks/cache --outpkg mocks --with-expecter --filename TagCache.go
type TagCache interface {
	GetTagsByPost(ctx context.Context, postID int64) ([]*model.Tag, error)
	SetTagsByPost(ctx context.Context, postID int64, tags []*model.Tag) error
	DeleteTagsByPost(ctx context.Context, postID int64) error
	GetTag(ctx context.Context, name string) (*model.Tag, error)
	SetTag(ctx context.Context, tag *model.Tag) error
	DeleteTag(ctx context.Context, name string) error
}
