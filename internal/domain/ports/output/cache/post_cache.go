package cache

import (
	"context"
	model "pinstack-post-service/internal/domain/models"
)

//go:generate mockery --name PostCache --dir . --output ../../../../mocks/cache --outpkg mocks --with-expecter --filename PostCache.go
type PostCache interface {
	GetPost(ctx context.Context, postID int64) (*model.PostDetailed, error)
	SetPost(ctx context.Context, post *model.PostDetailed) error
	DeletePost(ctx context.Context, postID int64) error
}
