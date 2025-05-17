package tag_repository_postgres

import (
	"context"
	"pinstack-post-service/internal/logger"
	"pinstack-post-service/internal/model"
	"pinstack-post-service/internal/repository"
)

type TagRepository struct {
	log *logger.Logger
	db  repository.DB
}

func NewTagRepository(db repository.DB, log *logger.Logger) *TagRepository {
	return &TagRepository{db: db, log: log}
}

func (t *TagRepository) FindByNames(ctx context.Context, names []string) ([]*model.PostTag, error) {
	//TODO implement me
	panic("implement me")
}

func (t *TagRepository) FindByPost(ctx context.Context, postID int) ([]*model.PostTag, error) {
	//TODO implement me
	panic("implement me")
}

func (t *TagRepository) FindPopular(ctx context.Context, limit int) ([]*model.PostTag, error) {
	//TODO implement me
	panic("implement me")
}

func (t *TagRepository) Create(ctx context.Context, name string) (*model.PostTag, error) {
	//TODO implement me
	panic("implement me")
}

func (t *TagRepository) DeleteUnused(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (t *TagRepository) TagPost(ctx context.Context, postID int, tagNames []string) error {
	//TODO implement me
	panic("implement me")
}

func (t *TagRepository) UntagPost(ctx context.Context, postID int, tagNames []string) error {
	//TODO implement me
	panic("implement me")
}

func (t *TagRepository) ReplacePostTags(ctx context.Context, postID int, newTags []string) error {
	//TODO implement me
	panic("implement me")
}
