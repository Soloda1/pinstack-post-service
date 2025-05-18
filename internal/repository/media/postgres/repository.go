package media_repository_postgres

import (
	"context"
	"pinstack-post-service/internal/logger"
	"pinstack-post-service/internal/model"
	"pinstack-post-service/internal/repository/postgres"
)

type MediaRepository struct {
	log *logger.Logger
	db  postgres.PgDB
}

func NewMediaRepository(db postgres.PgDB, log *logger.Logger) *MediaRepository {
	return &MediaRepository{db: db, log: log}
}

func (m *MediaRepository) Attach(ctx context.Context, postID int64, media []*model.PostMedia) error {
	//TODO implement me
	panic("implement me")
}

func (m *MediaRepository) Reorder(ctx context.Context, postID int64, newPositions map[int64]int) error {
	//TODO implement me
	panic("implement me")
}

func (m *MediaRepository) Detach(ctx context.Context, mediaIDs []int64) error {
	//TODO implement me
	panic("implement me")
}

func (m *MediaRepository) GetByPost(ctx context.Context, postID int64) ([]*model.PostMedia, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MediaRepository) GetByPosts(ctx context.Context, postIDs []int64) (map[int64][]*model.PostMedia, error) {
	//TODO implement me
	panic("implement me")
}
