package media_repository

import (
	"context"
	"pinstack-post-service/internal/domain/models"
)

//go:generate mockery --name Repository --dir . --output ../../../mocks --outpkg mocks --with-expecter --filename MediaRepository.go
type Repository interface {
	Attach(ctx context.Context, postID int64, media []*model.PostMedia) error
	Reorder(ctx context.Context, postID int64, newPositions map[int64]int) error
	Detach(ctx context.Context, mediaIDs []int64) error
	GetByPost(ctx context.Context, postID int64) ([]*model.PostMedia, error)
	GetByPosts(ctx context.Context, postIDs []int64) (map[int64][]*model.PostMedia, error)
}
