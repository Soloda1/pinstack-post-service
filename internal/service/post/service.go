package post_service

import (
	"context"
	"pinstack-post-service/internal/logger"
	"pinstack-post-service/internal/model"
	media_repository "pinstack-post-service/internal/repository/media"
	post_repository "pinstack-post-service/internal/repository/post"
	"pinstack-post-service/internal/repository/postgres"
	tag_repository "pinstack-post-service/internal/repository/tag"
)

type PostService struct {
	postRepo  post_repository.Repository
	tagRepo   tag_repository.Repository
	mediaRepo media_repository.Repository
	uow       postgres.UnitOfWork
	log       *logger.Logger
}

func NewPostService(
	postRepo post_repository.Repository,
	tagRepo tag_repository.Repository,
	mediaRepo media_repository.Repository,
	uow postgres.UnitOfWork,
	log *logger.Logger,
) *PostService {
	return &PostService{
		postRepo:  postRepo,
		tagRepo:   tagRepo,
		mediaRepo: mediaRepo,
		uow:       uow,
		log:       log,
	}
}

func (s *PostService) CreatePost(ctx context.Context, post *model.CreatePostDTO) (*model.Post, error) {
	panic("implement")
}

func (s *PostService) GetPostByID(ctx context.Context, id int64) (*model.PostDetailed, error) {
	panic("implement")
}

func (s *PostService) ListPosts(ctx context.Context, filters *model.PostFilters) ([]*model.PostDetailed, error) {
	panic("implement")
}

func (s *PostService) UpdatePost(ctx context.Context, id int64, post *model.UpdatePostDTO) error {
	panic("implement")
}

func (s *PostService) DeletePost(ctx context.Context, id int64) error {
	panic("implement")
}
