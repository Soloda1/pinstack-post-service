package post_service

import (
	"context"
	"errors"
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
	userClient user_client.Client
}

func NewPostService(
	postRepo post_repository.Repository,
	tagRepo tag_repository.Repository,
	mediaRepo media_repository.Repository,
	uow postgres.UnitOfWork,
	log *logger.Logger,
	userClient user_client.Client,
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
		if errors.Is(err, custom_errors.ErrDatabaseQuery) {
			s.log.Error("Database error in create post", slog.String("error", err.Error()))
			return nil, custom_errors.ErrDatabaseQuery
		}
		s.log.Error("Failed to create post", slog.String("error", err.Error()))
		return nil, err
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
				if errors.Is(err, custom_errors.ErrTagAlreadyExists) {
					s.log.Debug("Tag already exists", slog.String("error", err.Error()))
					return nil, custom_errors.ErrTagAlreadyExists
				}
				if errors.Is(err, custom_errors.ErrTagCreateFailed) {
					s.log.Error("Failed to create tag", slog.String("error", err.Error()))
					return nil, custom_errors.ErrTagCreateFailed
				}
				s.log.Error("Unknown error while creating tag", slog.String("error", err.Error()))
				return nil, err
			}
			createdTags = append(createdTags, createdTag)
		}

		err = tagRepo.TagPost(ctx, createdPost.ID, post.Tags)
		if err != nil {
			if errors.Is(err, custom_errors.ErrPostNotFound) {
				s.log.Debug("Post not found when adding tags", slog.String("error", err.Error()))
				return nil, custom_errors.ErrPostNotFound
			}
			if errors.Is(err, custom_errors.ErrTagNotFound) {
				s.log.Debug("Tag not found when adding to post", slog.String("error", err.Error()))
				return nil, custom_errors.ErrTagNotFound
			}
			if errors.Is(err, custom_errors.ErrTagVerifyPostFailed) {
				s.log.Error("Tag verification failed when adding tags to post", slog.String("error", err.Error()))
				return nil, custom_errors.ErrTagVerifyPostFailed
			}
			if errors.Is(err, custom_errors.ErrTagPost) {
				s.log.Error("Failed to add tags to post", slog.String("error", err.Error()))
				return nil, custom_errors.ErrTagPost
			}
			s.log.Error("Unknown error while adding tags to post", slog.String("error", err.Error()))
			return nil, err
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
	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, custom_errors.ErrPostNotFound):
			s.log.Debug("Post not found", slog.Int64("id", id))
			return nil, custom_errors.ErrPostNotFound
		default:
			s.log.Error("Failed to get post by id",
				slog.String("error", err.Error()),
				slog.Int64("id", id))
			return nil, custom_errors.ErrDatabaseQuery
		}
	}

	media, err := s.mediaRepo.GetByPost(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, custom_errors.ErrMediaNotFound):
			s.log.Debug("Media not found for post", slog.Int64("id", id))
			return nil, custom_errors.ErrMediaNotFound
		default:
			s.log.Error("Failed to get media by post",
				slog.String("error", err.Error()),
				slog.Int64("id", id))
			return nil, custom_errors.ErrDatabaseQuery
		}
	}

	tags, err := s.tagRepo.FindByPost(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, custom_errors.ErrTagsNotFound):
			s.log.Debug("Tags not found for post", slog.Int64("id", id))
			return nil, custom_errors.ErrTagsNotFound
		default:
			s.log.Error("Failed to find tags by post",
				slog.String("error", err.Error()),
				slog.Int64("id", id))
			return nil, custom_errors.ErrDatabaseQuery
		}
	}

	author, err := s.userClient.GetUser(ctx, post.AuthorID)
	if err != nil {
		switch {
		case errors.Is(err, custom_errors.ErrUserNotFound):
			s.log.Debug("Author not found", slog.Int64("authorID", post.AuthorID))
			return nil, custom_errors.ErrUserNotFound
		default:
			s.log.Error("Failed to get author",
				slog.String("error", err.Error()),
				slog.Int64("authorID", post.AuthorID))
			return nil, custom_errors.ErrDatabaseQuery
		}
	}

	postDetailed := &model.PostDetailed{
		Post:   post,
		Author: author,
		Media:  media,
		Tags:   tags,
	}
	return postDetailed, nil
}

func (s *PostService) ListPosts(ctx context.Context, filters *model.PostFilters) ([]*model.PostDetailed, error) {
	posts, err := s.postRepo.List(ctx, *filters)
	if err != nil {
		s.log.Error("Failed to list posts", slog.String("error", err.Error()))
		return nil, custom_errors.ErrDatabaseQuery
	}

	result := make([]*model.PostDetailed, 0, len(posts))
	for _, post := range posts {
		media, err := s.mediaRepo.GetByPost(ctx, post.ID)
		if err != nil {
			switch {
			case errors.Is(err, custom_errors.ErrMediaNotFound):
				s.log.Debug("Media not found for post", slog.Int64("id", post.ID))
				media = nil
			default:
				s.log.Error("Failed to get media by post", slog.String("error", err.Error()), slog.Int64("id", post.ID))
				return nil, custom_errors.ErrDatabaseQuery
			}
		}

		tags, err := s.tagRepo.FindByPost(ctx, post.ID)
		if err != nil {
			switch {
			case errors.Is(err, custom_errors.ErrTagsNotFound):
				s.log.Debug("Tags not found for post", slog.Int64("id", post.ID))
				tags = nil
			default:
				s.log.Error("Failed to find tags by post", slog.String("error", err.Error()), slog.Int64("id", post.ID))
				return nil, custom_errors.ErrDatabaseQuery
			}
		}

		author, err := s.userClient.GetUser(ctx, post.AuthorID)
		if err != nil {
			switch {
			case errors.Is(err, custom_errors.ErrUserNotFound):
				s.log.Debug("Author not found", slog.Int64("authorID", post.AuthorID))
				return nil, custom_errors.ErrUserNotFound
			default:
				s.log.Error("Failed to get author", slog.String("error", err.Error()), slog.Int64("authorID", post.AuthorID))
				return nil, custom_errors.ErrDatabaseQuery
			}
		}

		postDetailed := &model.PostDetailed{
			Post:   post,
			Author: author,
			Media:  media,
			Tags:   tags,
		}
		result = append(result, postDetailed)
	}
	return result, nil
}

func (s *PostService) UpdatePost(ctx context.Context, id int64, post *model.UpdatePostDTO) error {
	tx, err := s.uow.Begin(ctx)
	if err != nil {
		s.log.Error("Failed to start transaction", slog.String("error", err.Error()))
		return custom_errors.ErrDatabaseQuery
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	postRepo := tx.PostRepository()
	mediaRepo := tx.MediaRepository()
	tagRepo := tx.TagRepository()

	_, err = postRepo.Update(ctx, id, post)
	if err != nil {
		if errors.Is(err, custom_errors.ErrPostNotFound) {
			s.log.Debug("Post not found for update", slog.Int64("id", id))
			return custom_errors.ErrPostNotFound
		}
		s.log.Error("Failed to update post", slog.String("error", err.Error()), slog.Int64("id", id))
		return custom_errors.ErrDatabaseQuery
	}

	if post.MediaItems != nil {
		media, err := mediaRepo.GetByPost(ctx, id)
		if err != nil {
			if errors.Is(err, custom_errors.ErrMediaNotFound) {
				s.log.Debug("Media not found for update", slog.Int64("id", id))
				return custom_errors.ErrMediaNotFound
			}
			s.log.Error("Failed to get post media", slog.String("error", err.Error()), slog.Int64("id", id))
			return custom_errors.ErrDatabaseQuery
		}
		mediaIds := make([]int64, 0, len(media))
		for _, mediaItem := range media {
			mediaIds = append(mediaIds, mediaItem.ID)
		}
		err = mediaRepo.Detach(ctx, mediaIds)
		if err != nil {
			s.log.Error("Failed to clear media for post", slog.String("error", err.Error()), slog.Int64("id", id))
			return custom_errors.ErrMediaAttachFailed
		}
		if len(post.MediaItems) > 0 {
			media := make([]*model.PostMedia, 0, len(post.MediaItems))
			for _, m := range post.MediaItems {
				media = append(media, &model.PostMedia{
					PostID:   id,
					URL:      m.URL,
					Type:     m.Type,
					Position: m.Position,
				})
			}
			err = mediaRepo.Attach(ctx, id, media)
			if err != nil {
				s.log.Error("Failed to attach media to post", slog.String("error", err.Error()), slog.Int64("id", id))
				return custom_errors.ErrMediaAttachFailed
			}
		}
	}

	if post.Tags != nil && len(post.Tags) > 0 {
		for _, name := range post.Tags {
			_, err := tagRepo.Create(ctx, name)
			if err != nil && !errors.Is(err, custom_errors.ErrTagAlreadyExists) {
				if errors.Is(err, custom_errors.ErrTagCreateFailed) {
					s.log.Error("Failed to create tag", slog.String("error", err.Error()))
					return custom_errors.ErrTagCreateFailed
				}
				s.log.Error("Unknown error creating tag", slog.String("error", err.Error()))
				return err
			}
		}
		err = tagRepo.ReplacePostTags(ctx, id, post.Tags)
		if err != nil {
			if errors.Is(err, custom_errors.ErrPostNotFound) {
				s.log.Debug("Post not found when tagging", slog.String("error", err.Error()))
				return custom_errors.ErrPostNotFound
			}
			if errors.Is(err, custom_errors.ErrTagNotFound) {
				s.log.Debug("Tag not found when tagging post", slog.String("error", err.Error()))
				return custom_errors.ErrTagNotFound
			}
			if errors.Is(err, custom_errors.ErrTagVerifyPostFailed) {
				s.log.Error("Tag verify post failed", slog.String("error", err.Error()))
				return custom_errors.ErrTagVerifyPostFailed
			}
			if errors.Is(err, custom_errors.ErrTagPost) {
				s.log.Error("Failed to tag post", slog.String("error", err.Error()))
				return custom_errors.ErrTagPost
			}
			s.log.Error("Unknown error tagging post", slog.String("error", err.Error()))
			return err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		s.log.Error("Failed to commit transaction", slog.String("error", err.Error()))
		return custom_errors.ErrDatabaseQuery
	}

	return nil
}

func (s *PostService) DeletePost(ctx context.Context, userID int64, id int64) error {
	tx, err := s.uow.Begin(ctx)
	if err != nil {
		s.log.Error("Failed to start transaction", slog.String("error", err.Error()))
		return custom_errors.ErrDatabaseQuery
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	postRepo := tx.PostRepository()
	mediaRepo := tx.MediaRepository()
	tagRepo := tx.TagRepository()

	post, err := postRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, custom_errors.ErrPostNotFound) {
			s.log.Debug("Post not found when deleting post", slog.String("error", err.Error()))
			return custom_errors.ErrPostNotFound
		} else {
			s.log.Error("Failed to get post", slog.String("error", err.Error()), slog.Int64("id", id))
			return custom_errors.ErrDatabaseQuery
		}
	}
	if post.AuthorID != userID {
		s.log.Debug("User is not author of post", slog.String("error", err.Error()))
		return custom_errors.ErrInvalidInput
	}

	media, err := mediaRepo.GetByPost(ctx, id)
	if err != nil {
		if errors.Is(err, custom_errors.ErrMediaNotFound) {
			s.log.Debug("Media not found for post during delete", slog.Int64("id", id))
			media = nil
		} else {
			s.log.Error("Failed to get media for post during delete", slog.String("error", err.Error()), slog.Int64("id", id))
			return custom_errors.ErrMediaQueryFailed
		}
	}
	mediaIds := make([]int64, 0, len(media))
	for _, mediaItem := range media {
		mediaIds = append(mediaIds, mediaItem.ID)
	}
	if len(mediaIds) > 0 {
		err = mediaRepo.Detach(ctx, mediaIds)
		if err != nil {
			if errors.Is(err, custom_errors.ErrMediaNotFound) {
				s.log.Debug("Media not found for post during detach", slog.Int64("id", id))
			} else {
				s.log.Error("Failed to detach media for post", slog.String("error", err.Error()), slog.Int64("id", id))
				return custom_errors.ErrMediaDetachFailed
			}
		}
	}

	tags, err := tagRepo.FindByPost(ctx, id)
	if err != nil {
		if errors.Is(err, custom_errors.ErrTagsNotFound) {
			s.log.Debug("Tags not found for post during delete", slog.Int64("id", id))
			tags = nil
		} else {
			s.log.Error("Failed to get tags for post during delete", slog.String("error", err.Error()), slog.Int64("id", id))
			return custom_errors.ErrTagQueryFailed
		}
	}
	tagNames := make([]string, 0, len(tags))
	for _, tag := range tags {
		tagNames = append(tagNames, tag.Name)
	}
	if len(tagNames) > 0 {
		err = tagRepo.UntagPost(ctx, id, tagNames)
		if err != nil {
			if errors.Is(err, custom_errors.ErrTagNotFound) {
				s.log.Debug("Tags not found for post during untag", slog.Int64("id", id))
			} else {
				s.log.Error("Failed to untag post", slog.String("error", err.Error()), slog.Int64("id", id))
				return custom_errors.ErrTagDeleteFailed
			}
		}
	}
	err = postRepo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, custom_errors.ErrPostNotFound) {
			s.log.Debug("Post not found for delete", slog.Int64("id", id))
			return custom_errors.ErrPostNotFound
		}
		s.log.Error("Failed to delete post", slog.String("error", err.Error()), slog.Int64("id", id))
		return custom_errors.ErrDatabaseQuery
	}
	err = tx.Commit(ctx)
	if err != nil {
		s.log.Error("Failed to commit transaction", slog.String("error", err.Error()))
		return custom_errors.ErrDatabaseQuery
	}
	return nil
}
