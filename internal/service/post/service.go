package post_service

import (
	"context"
	"log/slog"
	user_client "pinstack-post-service/internal/clients/user"
	"pinstack-post-service/internal/custom_errors"
	"pinstack-post-service/internal/logger"
	"pinstack-post-service/internal/model"
	media_repository "pinstack-post-service/internal/repository/media"
	post_repository "pinstack-post-service/internal/repository/post"
	"pinstack-post-service/internal/repository/postgres"
	tag_repository "pinstack-post-service/internal/repository/tag"
	"sync"
)

type PostService struct {
	postRepo   post_repository.Repository
	tagRepo    tag_repository.Repository
	mediaRepo  media_repository.Repository
	uow        postgres.UnitOfWork
	log        *logger.Logger
	userClient user_client.UserClient
}

func NewPostService(
	postRepo post_repository.Repository,
	tagRepo tag_repository.Repository,
	mediaRepo media_repository.Repository,
	uow postgres.UnitOfWork,
	log *logger.Logger,
	userClient user_client.UserClient,
) *PostService {
	return &PostService{
		postRepo:   postRepo,
		tagRepo:    tagRepo,
		mediaRepo:  mediaRepo,
		uow:        uow,
		log:        log,
		userClient: userClient,
	}
}

func (s *PostService) CreatePost(ctx context.Context, post *model.CreatePostDTO) (*model.PostDetailed, error) {
	var author *model.User
	var userErr error
	wg := &sync.WaitGroup{}
	createdTags := make([]*model.Tag, 0, len(post.Tags))
	createdMedia := make([]*model.PostMedia, 0, len(post.MediaItems))

	wg.Add(1)
	go func() {
		defer wg.Done()
		author, userErr = s.userClient.GetUser(ctx, post.AuthorID)
		if userErr != nil {
			s.log.Error("Failed to get author from user service", slog.String("error", userErr.Error()))
			userErr = custom_errors.ErrExternalServiceError
			return
		}
	}()

	tx, err := s.uow.Begin(ctx)
	if err != nil {
		s.log.Error("Failed to start transaction", slog.String("error", err.Error()))
		return nil, custom_errors.ErrDatabaseQuery
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	postRepo := tx.PostRepository()
	mediaRepo := tx.MediaRepository()
	tagRepo := tx.TagRepository()

	newPost := &model.Post{
		AuthorID: post.AuthorID,
		Title:    post.Title,
		Content:  post.Content,
	}
	createdPost, err := postRepo.Create(ctx, newPost)
	if err != nil {
		s.log.Error("Failed to create post", slog.String("error", err.Error()))
		return nil, custom_errors.ErrDatabaseQuery
	}

	if post.MediaItems != nil && len(post.MediaItems) > 0 {
		media := make([]*model.PostMedia, 0, len(post.MediaItems))
		for _, m := range post.MediaItems {
			media = append(media, &model.PostMedia{
				PostID:   createdPost.ID,
				URL:      m.URL,
				Type:     m.Type,
				Position: m.Position,
			})
		}
		err = mediaRepo.Attach(ctx, createdPost.ID, media)
		if err != nil {
			s.log.Error("Failed to attach media to post", slog.String("error", err.Error()))
			return nil, custom_errors.ErrMediaAttachFailed
		}
		createdMedia, err = mediaRepo.GetByPost(ctx, createdPost.ID)
		if err != nil {
			s.log.Error("Failed to get media by post", slog.String("error", err.Error()))
			return nil, custom_errors.ErrMediaQueryFailed
		}
	}

	if post.Tags != nil && len(post.Tags) > 0 {
		for _, name := range post.Tags {
			createdTag, err := tagRepo.Create(ctx, name)
			if err != nil {
				s.log.Error("Failed to create tag", slog.String("error", err.Error()))
				return nil, custom_errors.ErrTagCreateFailed
			}
			createdTags = append(createdTags, createdTag)
		}

		err = tagRepo.TagPost(ctx, createdPost.ID, post.Tags)
		if err != nil {
			s.log.Error("Failed to add tags to post", slog.String("error", err.Error()))
			return nil, custom_errors.ErrTagPost
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		s.log.Error("Failed to commit transaction", slog.String("error", err.Error()))
		return nil, custom_errors.ErrDatabaseQuery
	}

	wg.Wait()

	if userErr != nil {
		return nil, userErr
	}

	postDetailed := &model.PostDetailed{
		Post:   createdPost,
		Author: author,
		Media:  createdMedia,
		Tags:   createdTags,
	}
	return postDetailed, nil
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
