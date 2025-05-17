package media_repository_postgres

import (
	"context"
	"pinstack-post-service/internal/logger"
	"pinstack-post-service/internal/model"
	"pinstack-post-service/internal/repository"
)

type MediaRepository struct {
	log *logger.Logger
	db  repository.DB
}

func NewMediaRepository(db repository.DB, log *logger.Logger) *MediaRepository {
	return &MediaRepository{db: db, log: log}
}

func (m *MediaRepository) Attach(ctx context.Context, postID int, media []*model.PostMedia) error {
	//TODO implement me
	panic("implement me")
}

func (m *MediaRepository) Reorder(ctx context.Context, postID int, newPositions map[int]int) error {
	//TODO implement me
	panic("implement me")
}

func (m *MediaRepository) Detach(ctx context.Context, mediaIDs []int) error {
	//TODO implement me
	panic("implement me")
}

func (m *MediaRepository) GetByPost(ctx context.Context, postID int) ([]*model.PostMedia, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MediaRepository) GetByPosts(ctx context.Context, postIDs []int) (map[int][]*model.PostMedia, error) {
	//TODO implement me
	panic("implement me")
}
