package media_repository

import (
	"context"
	"pinstack-post-service/internal/model"
)

type Repository interface {
	Attach(ctx context.Context, postID int, media []*model.PostMedia) error
	Reorder(ctx context.Context, postID int, newPositions map[int]int) error
	Detach(ctx context.Context, mediaIDs []int) error
	GetByPost(ctx context.Context, postID int) ([]*model.PostMedia, error)
	GetByPosts(ctx context.Context, postIDs []int) (map[int][]*model.PostMedia, error)
}
