package post_repository_postgres

import (
	"context"
	"pinstack-post-service/internal/logger"
	"pinstack-post-service/internal/model"
	"pinstack-post-service/internal/repository"
)

type PostRepository struct {
	log *logger.Logger
	db  repository.DB
}

func NewPostRepository(db repository.DB, log *logger.Logger) *PostRepository {
	return &PostRepository{db: db, log: log}
}

func (p *PostRepository) Create(ctx context.Context, post *model.Post) error {
	//TODO implement me
	panic("implement me")
}

func (p *PostRepository) GetByID(ctx context.Context, id int) (*model.Post, error) {
	//TODO implement me
	panic("implement me")
}

func (p *PostRepository) GetByAuthor(ctx context.Context, authorID int64) ([]*model.Post, error) {
	//TODO implement me
	panic("implement me")
}

func (p *PostRepository) UpdateTitle(ctx context.Context, id int, title string) error {
	//TODO implement me
	panic("implement me")
}

func (p *PostRepository) UpdateContent(ctx context.Context, id int, content string) error {
	//TODO implement me
	panic("implement me")
}

func (p *PostRepository) Delete(ctx context.Context, id int) error {
	//TODO implement me
	panic("implement me")
}

func (p *PostRepository) List(ctx context.Context, filters model.PostFilters) ([]*model.Post, error) {
	//TODO implement me
	panic("implement me")
}
