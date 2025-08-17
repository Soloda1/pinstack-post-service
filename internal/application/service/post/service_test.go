package post_service

import (
	"context"
	"errors"
	"pinstack-post-service/internal/infrastructure/outbound/metrics/prometheus"
	"testing"

	"github.com/soloda1/pinstack-proto-definitions/custom_errors"

	user_client_mock "pinstack-post-service/mocks/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	model "pinstack-post-service/internal/domain/models"
	"pinstack-post-service/internal/infrastructure/logger"
	media_repository_mock "pinstack-post-service/mocks/media"
	post_repository_mock "pinstack-post-service/mocks/post"
	postgres_mock "pinstack-post-service/mocks/postgres"
	tag_repository_mock "pinstack-post-service/mocks/tag"
)

func TestPostService_CreatePost(t *testing.T) {
	log := logger.New("test")
	type args struct {
		ctx  context.Context
		post *model.CreatePostDTO
	}
	tests := []struct {
		name        string
		mocks       func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, userClient *user_client_mock.Client, tx *postgres_mock.Transaction)
		args        args
		want        *model.PostDetailed
		wantErr     bool
		wantErrType error
	}{
		{
			name: "Success",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, userClient *user_client_mock.Client, tx *postgres_mock.Transaction) {
				userClient.On("GetUser", mock.Anything, int64(1)).Return(&model.User{ID: 1, Username: "testuser"}, nil)
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)
				tx.On("TagRepository").Return(tagRepo)
				postRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Post")).Return(&model.Post{ID: 1, AuthorID: 1, Title: "Test Post"}, nil)
				mediaRepo.On("Attach", mock.Anything, int64(1), mock.AnythingOfType("[]*model.PostMedia")).Return(nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return([]*model.PostMedia{{ID: 1, PostID: 1, URL: "http://example.com/image.jpg", Type: model.MediaTypeImage, Position: 1}}, nil)
				tagRepo.On("FindByNames", mock.Anything, []string{"tag1", "tag2"}).Return([]*model.Tag{{ID: 1, Name: "tag1"}}, nil)
				tagRepo.On("Create", mock.Anything, "tag2").Return(&model.Tag{ID: 2, Name: "tag2"}, nil)
				tagRepo.On("TagPost", mock.Anything, int64(1), []string{"tag1", "tag2"}).Return(nil)
				tx.On("Commit", mock.Anything).Return(nil)
			},
			args: args{
				ctx: context.Background(),
				post: &model.CreatePostDTO{
					AuthorID: 1,
					Title:    "Test Post",
					Content:  func() *string { s := "Test content"; return &s }(),
					Tags:     []string{"tag1", "tag2"},
					MediaItems: []*model.PostMediaInput{
						{
							URL:      "http://example.com/image.jpg",
							Type:     model.MediaTypeImage,
							Position: 1,
						},
					},
				},
			},
			want: &model.PostDetailed{
				Post:   &model.Post{ID: 1, AuthorID: 1, Title: "Test Post"},
				Author: &model.User{ID: 1, Username: "testuser"},
				Media:  []*model.PostMedia{{ID: 1, PostID: 1, URL: "http://example.com/image.jpg", Type: model.MediaTypeImage, Position: 1}},
				Tags:   []*model.Tag{{ID: 1, Name: "tag1"}, {ID: 2, Name: "tag2"}},
			},
			wantErr: false,
		},
		{
			name: "Error getting user",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, userClient *user_client_mock.Client, tx *postgres_mock.Transaction) {
				userClient.On("GetUser", mock.Anything, int64(1)).Return(nil, assert.AnError)
				// With the fix in service.go, if GetUser fails, the function should return before starting a transaction.
			},
			args: args{
				ctx: context.Background(),
				post: &model.CreatePostDTO{
					AuthorID: 1, // Match the GetUser mock
					Title:    "Test Post for GetUser Error",
					Content:  func() *string { s := "Test content"; return &s }(),
				},
			},
			wantErr:     true,
			wantErrType: custom_errors.ErrExternalServiceError,
		},
		{
			name: "Transaction begin error",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, userClient *user_client_mock.Client, tx *postgres_mock.Transaction) {
				userClient.On("GetUser", mock.Anything, int64(1)).Return(&model.User{ID: 1, Username: "testuser"}, nil)
				uow.On("Begin", mock.Anything).Return(nil, errors.New("db error"))
			},
			args: args{
				ctx: context.Background(),
				post: &model.CreatePostDTO{
					AuthorID: 1,
					Title:    "Test Post",
				},
			},
			want:        nil,
			wantErr:     true,
			wantErrType: custom_errors.ErrDatabaseQuery,
		},
		{
			name: "Error creating post in repository",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, userClient *user_client_mock.Client, tx *postgres_mock.Transaction) {
				userClient.On("GetUser", mock.Anything, int64(1)).Return(&model.User{ID: 1, Username: "testuser"}, nil)
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo) // Needed for defer
				tx.On("TagRepository").Return(tagRepo)     // Needed for defer
				postRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Post")).Return(nil, custom_errors.ErrDatabaseQuery)
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx: context.Background(),
				post: &model.CreatePostDTO{
					AuthorID: 1,
					Title:    "Test Post",
				},
			},
			want:        nil,
			wantErr:     true,
			wantErrType: custom_errors.ErrDatabaseQuery,
		},
		{
			name: "Error attaching media",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, userClient *user_client_mock.Client, tx *postgres_mock.Transaction) {
				userClient.On("GetUser", mock.Anything, int64(1)).Return(&model.User{ID: 1, Username: "testuser"}, nil)
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)
				tx.On("TagRepository").Return(tagRepo) // Needed for defer
				postRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Post")).Return(&model.Post{ID: 1, AuthorID: 1, Title: "Test Post"}, nil)
				mediaRepo.On("Attach", mock.Anything, int64(1), mock.AnythingOfType("[]*model.PostMedia")).Return(custom_errors.ErrMediaAttachFailed)
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx: context.Background(),
				post: &model.CreatePostDTO{
					AuthorID:   1,
					Title:      "Test Post",
					MediaItems: []*model.PostMediaInput{{URL: "url", Type: "image", Position: 1}},
				},
			},
			want:        nil,
			wantErr:     true,
			wantErrType: custom_errors.ErrMediaAttachFailed,
		},
		{
			name: "Error getting media after attach",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, userClient *user_client_mock.Client, tx *postgres_mock.Transaction) {
				userClient.On("GetUser", mock.Anything, int64(1)).Return(&model.User{ID: 1, Username: "testuser"}, nil)
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)
				tx.On("TagRepository").Return(tagRepo) // Needed for defer
				postRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Post")).Return(&model.Post{ID: 1, AuthorID: 1, Title: "Test Post"}, nil)
				mediaRepo.On("Attach", mock.Anything, int64(1), mock.AnythingOfType("[]*model.PostMedia")).Return(nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return(nil, custom_errors.ErrMediaQueryFailed)
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx: context.Background(),
				post: &model.CreatePostDTO{
					AuthorID:   1,
					Title:      "Test Post",
					MediaItems: []*model.PostMediaInput{{URL: "url", Type: "image", Position: 1}},
				},
			},
			want:        nil,
			wantErr:     true,
			wantErrType: custom_errors.ErrMediaQueryFailed,
		},
		{
			name: "Error finding existing tags",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, userClient *user_client_mock.Client, tx *postgres_mock.Transaction) {
				userClient.On("GetUser", mock.Anything, int64(1)).Return(&model.User{ID: 1, Username: "testuser"}, nil)
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)
				tx.On("TagRepository").Return(tagRepo)
				postRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Post")).Return(&model.Post{ID: 1, AuthorID: 1, Title: "Test Post"}, nil)
				// No media for simplicity in this tag-focused error case
				tagRepo.On("FindByNames", mock.Anything, []string{"tag1"}).Return(nil, custom_errors.ErrTagQueryFailed)
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx: context.Background(),
				post: &model.CreatePostDTO{
					AuthorID: 1,
					Title:    "Test Post",
					Tags:     []string{"tag1"},
				},
			},
			want:        nil,
			wantErr:     true,
			wantErrType: custom_errors.ErrTagQueryFailed,
		},
		{
			name: "Error creating new tag",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, userClient *user_client_mock.Client, tx *postgres_mock.Transaction) {
				userClient.On("GetUser", mock.Anything, int64(1)).Return(&model.User{ID: 1, Username: "testuser"}, nil)
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)
				tx.On("TagRepository").Return(tagRepo)
				postRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Post")).Return(&model.Post{ID: 1, AuthorID: 1, Title: "Test Post"}, nil)
				tagRepo.On("FindByNames", mock.Anything, []string{"newtag"}).Return([]*model.Tag{}, nil) // No existing tags found
				tagRepo.On("Create", mock.Anything, "newtag").Return(nil, custom_errors.ErrTagCreateFailed)
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx: context.Background(),
				post: &model.CreatePostDTO{
					AuthorID: 1,
					Title:    "Test Post",
					Tags:     []string{"newtag"},
				},
			},
			want:        nil,
			wantErr:     true,
			wantErrType: custom_errors.ErrTagCreateFailed,
		},
		{
			name: "Error tagging post",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, userClient *user_client_mock.Client, tx *postgres_mock.Transaction) {
				userClient.On("GetUser", mock.Anything, int64(1)).Return(&model.User{ID: 1, Username: "testuser"}, nil)
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)
				tx.On("TagRepository").Return(tagRepo)
				postRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Post")).Return(&model.Post{ID: 1, AuthorID: 1, Title: "Test Post"}, nil)
				tagRepo.On("FindByNames", mock.Anything, []string{"tag1"}).Return([]*model.Tag{{ID: 1, Name: "tag1"}}, nil)
				tagRepo.On("TagPost", mock.Anything, int64(1), []string{"tag1"}).Return(custom_errors.ErrTagPost)
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx: context.Background(),
				post: &model.CreatePostDTO{
					AuthorID: 1,
					Title:    "Test Post",
					Tags:     []string{"tag1"},
				},
			},
			want:        nil,
			wantErr:     true,
			wantErrType: custom_errors.ErrTagPost,
		},
		{
			name: "Error committing transaction",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, userClient *user_client_mock.Client, tx *postgres_mock.Transaction) {
				userClient.On("GetUser", mock.Anything, int64(1)).Return(&model.User{ID: 1, Username: "testuser"}, nil)
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)
				tx.On("TagRepository").Return(tagRepo)
				postRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Post")).Return(&model.Post{ID: 1, AuthorID: 1, Title: "Test Post"}, nil)
				// Assuming no media and no new tags for simplicity in this commit-focused error case
				// FindByNames and TagPost are not called if post.Tags is nil
				tx.On("Commit", mock.Anything).Return(errors.New("commit error"))
				tx.On("Rollback", mock.Anything).Return(nil) // Rollback should still be called by defer if commit fails
			},
			args: args{
				ctx: context.Background(),
				post: &model.CreatePostDTO{
					AuthorID: 1,
					Title:    "Test Post",
				},
			},
			want:        nil,
			wantErr:     true,
			wantErrType: custom_errors.ErrDatabaseQuery,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postRepo := new(post_repository_mock.Repository)
			tagRepo := new(tag_repository_mock.Repository)
			mediaRepo := new(media_repository_mock.Repository)
			uow := new(postgres_mock.UnitOfWork)
			userClient := new(user_client_mock.Client)
			tx := new(postgres_mock.Transaction)
			metrics := prometheus.NewPrometheusMetricsProvider()

			if tt.mocks != nil {
				tt.mocks(postRepo, tagRepo, mediaRepo, uow, userClient, tx)
			}

			s := NewPostService(postRepo, tagRepo, mediaRepo, uow, log, userClient, metrics)
			got, err := s.CreatePost(tt.args.ctx, tt.args.post)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrType != nil {
					assert.True(t, errors.Is(err, tt.wantErrType), "expected error type %T, got %T", tt.wantErrType, err)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)

			postRepo.AssertExpectations(t)
			tagRepo.AssertExpectations(t)
			mediaRepo.AssertExpectations(t)
			uow.AssertExpectations(t)
			userClient.AssertExpectations(t)
			tx.AssertExpectations(t)
		})
	}
}

func TestPostService_GetPostByID(t *testing.T) {
	log := logger.New("test")
	type args struct {
		ctx    context.Context
		postID int64
	}
	tests := []struct {
		name        string
		mocks       func(postRepo *post_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, userClient *user_client_mock.Client)
		args        args
		want        *model.PostDetailed
		wantErr     bool
		wantErrType error
	}{
		{
			name: "Success",
			mocks: func(postRepo *post_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, userClient *user_client_mock.Client) {
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 1, Title: "Test Post"}, nil)
				userClient.On("GetUser", mock.Anything, int64(1)).Return(&model.User{ID: 1, Username: "testuser"}, nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return([]*model.PostMedia{{ID: 1, PostID: 1, URL: "url", Type: "image"}}, nil)
				tagRepo.On("FindByPost", mock.Anything, int64(1)).Return([]*model.Tag{{ID: 1, Name: "tag1"}}, nil)
			},
			args: args{
				ctx:    context.Background(),
				postID: 1,
			},
			want: &model.PostDetailed{
				Post:   &model.Post{ID: 1, AuthorID: 1, Title: "Test Post"},
				Author: &model.User{ID: 1, Username: "testuser"},
				Media:  []*model.PostMedia{{ID: 1, PostID: 1, URL: "url", Type: "image"}},
				Tags:   []*model.Tag{{ID: 1, Name: "tag1"}},
			},
			wantErr: false,
		},
		{
			name: "Error post not found",
			mocks: func(postRepo *post_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, userClient *user_client_mock.Client) {
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(nil, custom_errors.ErrPostNotFound)
			},
			args: args{
				ctx:    context.Background(),
				postID: 1,
			},
			want:        nil,
			wantErr:     true,
			wantErrType: custom_errors.ErrPostNotFound,
		},
		{
			name: "Error getting user",
			mocks: func(postRepo *post_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, userClient *user_client_mock.Client) {
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 1, Title: "Test Post"}, nil)
				userClient.On("GetUser", mock.Anything, int64(1)).Return(nil, errors.New("user service error"))
			},
			args: args{
				ctx:    context.Background(),
				postID: 1,
			},
			want:        nil,
			wantErr:     true,
			wantErrType: custom_errors.ErrExternalServiceError,
		},
		{
			name: "Error getting media",
			mocks: func(postRepo *post_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, userClient *user_client_mock.Client) {
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 1, Title: "Test Post"}, nil)
				userClient.On("GetUser", mock.Anything, int64(1)).Return(&model.User{ID: 1, Username: "testuser"}, nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return(nil, custom_errors.ErrMediaQueryFailed)
			},
			args: args{
				ctx:    context.Background(),
				postID: 1,
			},
			want:        nil,
			wantErr:     true,
			wantErrType: custom_errors.ErrMediaQueryFailed,
		},
		{
			name: "Error getting tags",
			mocks: func(postRepo *post_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, userClient *user_client_mock.Client) {
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 1, Title: "Test Post"}, nil)
				userClient.On("GetUser", mock.Anything, int64(1)).Return(&model.User{ID: 1, Username: "testuser"}, nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return([]*model.PostMedia{}, nil)
				tagRepo.On("FindByPost", mock.Anything, int64(1)).Return(nil, custom_errors.ErrTagQueryFailed)
			},
			args: args{
				ctx:    context.Background(),
				postID: 1,
			},
			want:        nil,
			wantErr:     true,
			wantErrType: custom_errors.ErrTagQueryFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postRepo := new(post_repository_mock.Repository)
			tagRepo := new(tag_repository_mock.Repository)
			mediaRepo := new(media_repository_mock.Repository)
			userClient := new(user_client_mock.Client)
			// uow is not used in GetPostByID, so we don't need to mock it here
			uow := new(postgres_mock.UnitOfWork)
			metrics := prometheus.NewPrometheusMetricsProvider()

			if tt.mocks != nil {
				tt.mocks(postRepo, mediaRepo, tagRepo, userClient)
			}

			s := NewPostService(postRepo, tagRepo, mediaRepo, uow, log, userClient, metrics)
			got, err := s.GetPostByID(tt.args.ctx, tt.args.postID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrType != nil {
					assert.True(t, errors.Is(err, tt.wantErrType), "expected error type %T, got %T", tt.wantErrType, err)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)

			postRepo.AssertExpectations(t)
			tagRepo.AssertExpectations(t)
			mediaRepo.AssertExpectations(t)
			userClient.AssertExpectations(t)
		})
	}
}

func TestPostService_ListPosts(t *testing.T) {
	log := logger.New("test")
	type args struct {
		ctx     context.Context
		filters *model.PostFilters
	}
	tests := []struct {
		name        string
		mocks       func(postRepo *post_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, userClient *user_client_mock.Client)
		args        args
		want        []*model.PostDetailed
		wantErr     bool
		wantErrType error
	}{
		{
			name: "Success",
			mocks: func(postRepo *post_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, userClient *user_client_mock.Client) {
				filters := model.PostFilters{Limit: func(i int) *int { return &i }(10), Offset: func(i int) *int { return &i }(0)}
				posts := []*model.Post{
					{ID: 1, AuthorID: 1, Title: "Post 1"},
					{ID: 2, AuthorID: 2, Title: "Post 2"},
				}
				postRepo.On("List", mock.Anything, filters).Return(posts, len(posts), nil)

				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return([]*model.PostMedia{{ID: 1, PostID: 1, URL: "url1", Type: "image"}}, nil)
				tagRepo.On("FindByPost", mock.Anything, int64(1)).Return([]*model.Tag{{ID: 1, Name: "tag1"}}, nil)
				userClient.On("GetUser", mock.Anything, int64(1)).Return(&model.User{ID: 1, Username: "user1"}, nil)

				mediaRepo.On("GetByPost", mock.Anything, int64(2)).Return([]*model.PostMedia{{ID: 2, PostID: 2, URL: "url2", Type: "video"}}, nil)
				tagRepo.On("FindByPost", mock.Anything, int64(2)).Return([]*model.Tag{{ID: 2, Name: "tag2"}}, nil)
				userClient.On("GetUser", mock.Anything, int64(2)).Return(&model.User{ID: 2, Username: "user2"}, nil)
			},
			args: args{
				ctx:     context.Background(),
				filters: &model.PostFilters{Limit: func(i int) *int { return &i }(10), Offset: func(i int) *int { return &i }(0)},
			},
			want: []*model.PostDetailed{
				{
					Post:   &model.Post{ID: 1, AuthorID: 1, Title: "Post 1"},
					Author: &model.User{ID: 1, Username: "user1"},
					Media:  []*model.PostMedia{{ID: 1, PostID: 1, URL: "url1", Type: "image"}},
					Tags:   []*model.Tag{{ID: 1, Name: "tag1"}},
				},
				{
					Post:   &model.Post{ID: 2, AuthorID: 2, Title: "Post 2"},
					Author: &model.User{ID: 2, Username: "user2"},
					Media:  []*model.PostMedia{{ID: 2, PostID: 2, URL: "url2", Type: "video"}},
					Tags:   []*model.Tag{{ID: 2, Name: "tag2"}},
				},
			},
			wantErr: false,
		},
		{
			name: "Error listing posts from repo",
			mocks: func(postRepo *post_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, userClient *user_client_mock.Client) {
				filters := model.PostFilters{Limit: func(i int) *int { return &i }(10), Offset: func(i int) *int { return &i }(0)}
				postRepo.On("List", mock.Anything, filters).Return(nil, 0, errors.New("db error"))
			},
			args: args{
				ctx:     context.Background(),
				filters: &model.PostFilters{Limit: func(i int) *int { return &i }(10), Offset: func(i int) *int { return &i }(0)},
			},
			want:        nil,
			wantErr:     true,
			wantErrType: custom_errors.ErrDatabaseQuery,
		},
		{
			name: "Error getting media for a post",
			mocks: func(postRepo *post_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, userClient *user_client_mock.Client) {
				filters := model.PostFilters{Limit: func(i int) *int { return &i }(10), Offset: func(i int) *int { return &i }(0)}
				posts := []*model.Post{{ID: 1, AuthorID: 1, Title: "Post 1"}}
				postRepo.On("List", mock.Anything, filters).Return(posts, len(posts), nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return(nil, errors.New("db error"))
			},
			args: args{
				ctx:     context.Background(),
				filters: &model.PostFilters{Limit: func(i int) *int { return &i }(10), Offset: func(i int) *int { return &i }(0)},
			},
			want:        nil,
			wantErr:     true,
			wantErrType: custom_errors.ErrDatabaseQuery,
		},
		{
			name: "Media not found for a post (should be nil in result)",
			mocks: func(postRepo *post_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, userClient *user_client_mock.Client) {
				filters := model.PostFilters{Limit: func(i int) *int { return &i }(10), Offset: func(i int) *int { return &i }(0)}
				posts := []*model.Post{{ID: 1, AuthorID: 1, Title: "Post 1"}}
				postRepo.On("List", mock.Anything, filters).Return(posts, len(posts), nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return(nil, custom_errors.ErrMediaNotFound)
				tagRepo.On("FindByPost", mock.Anything, int64(1)).Return([]*model.Tag{{ID: 1, Name: "tag1"}}, nil)
				userClient.On("GetUser", mock.Anything, int64(1)).Return(&model.User{ID: 1, Username: "user1"}, nil)
			},
			args: args{
				ctx:     context.Background(),
				filters: &model.PostFilters{Limit: func(i int) *int { return &i }(10), Offset: func(i int) *int { return &i }(0)},
			},
			want: []*model.PostDetailed{
				{
					Post:   &model.Post{ID: 1, AuthorID: 1, Title: "Post 1"},
					Author: &model.User{ID: 1, Username: "user1"},
					Media:  nil,
					Tags:   []*model.Tag{{ID: 1, Name: "tag1"}},
				},
			},
			wantErr: false,
		},
		{
			name: "Error getting tags for a post",
			mocks: func(postRepo *post_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, userClient *user_client_mock.Client) {
				filters := model.PostFilters{Limit: func(i int) *int { return &i }(10), Offset: func(i int) *int { return &i }(0)}
				posts := []*model.Post{{ID: 1, AuthorID: 1, Title: "Post 1"}}
				postRepo.On("List", mock.Anything, filters).Return(posts, len(posts), nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return([]*model.PostMedia{}, nil)
				tagRepo.On("FindByPost", mock.Anything, int64(1)).Return(nil, errors.New("db error"))
			},
			args: args{
				ctx:     context.Background(),
				filters: &model.PostFilters{Limit: func(i int) *int { return &i }(10), Offset: func(i int) *int { return &i }(0)},
			},
			want:        nil,
			wantErr:     true,
			wantErrType: custom_errors.ErrDatabaseQuery,
		},
		{
			name: "Tags not found for a post (should be nil in result)",
			mocks: func(postRepo *post_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, userClient *user_client_mock.Client) {
				filters := model.PostFilters{Limit: func(i int) *int { return &i }(10), Offset: func(i int) *int { return &i }(0)}
				posts := []*model.Post{{ID: 1, AuthorID: 1, Title: "Post 1"}}
				postRepo.On("List", mock.Anything, filters).Return(posts, len(posts), nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return([]*model.PostMedia{}, nil)
				tagRepo.On("FindByPost", mock.Anything, int64(1)).Return(nil, custom_errors.ErrTagsNotFound)
				userClient.On("GetUser", mock.Anything, int64(1)).Return(&model.User{ID: 1, Username: "user1"}, nil)
			},
			args: args{
				ctx:     context.Background(),
				filters: &model.PostFilters{Limit: func(i int) *int { return &i }(10), Offset: func(i int) *int { return &i }(0)},
			},
			want: []*model.PostDetailed{
				{
					Post:   &model.Post{ID: 1, AuthorID: 1, Title: "Post 1"},
					Author: &model.User{ID: 1, Username: "user1"},
					Media:  []*model.PostMedia{},
					Tags:   nil,
				},
			},
			wantErr: false,
		},
		{
			name: "Error getting user for a post",
			mocks: func(postRepo *post_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, userClient *user_client_mock.Client) {
				filters := model.PostFilters{Limit: func(i int) *int { return &i }(10), Offset: func(i int) *int { return &i }(0)}
				posts := []*model.Post{{ID: 1, AuthorID: 1, Title: "Post 1"}}
				postRepo.On("List", mock.Anything, filters).Return(posts, len(posts), nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return([]*model.PostMedia{}, nil)
				tagRepo.On("FindByPost", mock.Anything, int64(1)).Return([]*model.Tag{}, nil)
				userClient.On("GetUser", mock.Anything, int64(1)).Return(nil, errors.New("user service error"))
			},
			args: args{
				ctx:     context.Background(),
				filters: &model.PostFilters{Limit: func(i int) *int { return &i }(10), Offset: func(i int) *int { return &i }(0)},
			},
			want:        nil,
			wantErr:     true,
			wantErrType: custom_errors.ErrDatabaseQuery, // As per current service logic
		},
		{
			name: "User not found for a post",
			mocks: func(postRepo *post_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, userClient *user_client_mock.Client) {
				filters := model.PostFilters{Limit: func(i int) *int { return &i }(10), Offset: func(i int) *int { return &i }(0)}
				posts := []*model.Post{{ID: 1, AuthorID: 1, Title: "Post 1"}}
				postRepo.On("List", mock.Anything, filters).Return(posts, len(posts), nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return([]*model.PostMedia{}, nil)
				tagRepo.On("FindByPost", mock.Anything, int64(1)).Return([]*model.Tag{}, nil)
				userClient.On("GetUser", mock.Anything, int64(1)).Return(nil, custom_errors.ErrUserNotFound)
			},
			args: args{
				ctx:     context.Background(),
				filters: &model.PostFilters{Limit: func(i int) *int { return &i }(10), Offset: func(i int) *int { return &i }(0)},
			},
			want:        nil,
			wantErr:     true,
			wantErrType: custom_errors.ErrUserNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postRepo := new(post_repository_mock.Repository)
			tagRepo := new(tag_repository_mock.Repository)
			mediaRepo := new(media_repository_mock.Repository)
			userClient := new(user_client_mock.Client)
			uow := new(postgres_mock.UnitOfWork) // Not used directly in ListPosts but part of service struct
			metrics := prometheus.NewPrometheusMetricsProvider()

			if tt.mocks != nil {
				tt.mocks(postRepo, mediaRepo, tagRepo, userClient)
			}

			s := NewPostService(postRepo, tagRepo, mediaRepo, uow, log, userClient, metrics)
			got, total, err := s.ListPosts(tt.args.ctx, tt.args.filters)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrType != nil {
					assert.True(t, errors.Is(err, tt.wantErrType), "expected error type %T, got %T", tt.wantErrType, err)
				}
				assert.Nil(t, got)
				assert.Zero(t, total)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
			assert.Equal(t, tt.want, got)

			postRepo.AssertExpectations(t)
			tagRepo.AssertExpectations(t)
			mediaRepo.AssertExpectations(t)
			userClient.AssertExpectations(t)
		})
	}
}

func TestPostService_UpdatePost(t *testing.T) {
	log := logger.New("test")
	type args struct {
		ctx    context.Context
		userID int64
		postID int64
		post   *model.UpdatePostDTO
	}
	tests := []struct {
		name        string
		mocks       func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction)
		args        args
		wantErr     bool
		wantErrType error
	}{
		{
			name: "Success",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)
				tx.On("TagRepository").Return(tagRepo)

				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 1}, nil)
				postRepo.On("Update", mock.Anything, int64(1), mock.AnythingOfType("*model.UpdatePostDTO")).Return(&model.Post{}, nil)

				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return([]*model.PostMedia{{ID: 10}}, nil)
				mediaRepo.On("Detach", mock.Anything, []int64{10}).Return(nil)
				mediaRepo.On("Attach", mock.Anything, int64(1), mock.AnythingOfType("[]*model.PostMedia")).Return(nil)

				tagRepo.On("Create", mock.Anything, "newtag").Return(&model.Tag{ID: 1, Name: "newtag"}, nil)
				tagRepo.On("ReplacePostTags", mock.Anything, int64(1), []string{"newtag"}).Return(nil)

				tx.On("Commit", mock.Anything).Return(nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				postID: 1,
				post: &model.UpdatePostDTO{
					Title:      func() *string { s := "Updated Title"; return &s }(),
					MediaItems: []*model.PostMediaInput{{URL: "new_url", Type: "image", Position: 1}},
					Tags:       []string{"newtag"},
				},
			},
			wantErr: false,
		},
		{
			name: "Error begin transaction",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(nil, errors.New("db error"))
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				postID: 1,
				post:   &model.UpdatePostDTO{},
			},
			wantErr:     true,
			wantErrType: custom_errors.ErrDatabaseQuery,
		},
		{
			name: "Error GetByID post not found",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo) // For defer
				tx.On("TagRepository").Return(tagRepo)     // For defer
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(nil, custom_errors.ErrPostNotFound)
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				postID: 1,
				post:   &model.UpdatePostDTO{},
			},
			wantErr:     true,
			wantErrType: custom_errors.ErrPostNotFound,
		},
		{
			name: "Error user is not author",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)                                                   // For defer
				tx.On("TagRepository").Return(tagRepo)                                                       // For defer
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 2}, nil) // Different AuthorID
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 1, // UserID is 1
				postID: 1,
				post:   &model.UpdatePostDTO{},
			},
			wantErr:     true,
			wantErrType: custom_errors.ErrInvalidInput,
		},
		{
			name: "Error updating post in repo",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo) // For defer
				tx.On("TagRepository").Return(tagRepo)     // For defer
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 1}, nil)
				postRepo.On("Update", mock.Anything, int64(1), mock.AnythingOfType("*model.UpdatePostDTO")).Return(nil, custom_errors.ErrDatabaseQuery)
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				postID: 1,
				post:   &model.UpdatePostDTO{Title: func() *string { s := "t"; return &s }()},
			},
			wantErr:     true,
			wantErrType: custom_errors.ErrDatabaseQuery,
		},
		{
			name: "Error detaching media",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)
				tx.On("TagRepository").Return(tagRepo) // For defer
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 1}, nil)
				postRepo.On("Update", mock.Anything, int64(1), mock.AnythingOfType("*model.UpdatePostDTO")).Return(&model.Post{}, nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return([]*model.PostMedia{{ID: 10}}, nil)
				mediaRepo.On("Detach", mock.Anything, []int64{10}).Return(errors.New("detach error"))
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				postID: 1,
				post: &model.UpdatePostDTO{
					MediaItems: []*model.PostMediaInput{{URL: "new_url", Type: "image", Position: 1}},
				},
			},
			wantErr:     true,
			wantErrType: custom_errors.ErrMediaAttachFailed,
		},
		{
			name: "Error attaching media",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)
				tx.On("TagRepository").Return(tagRepo) // For defer
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 1}, nil)
				postRepo.On("Update", mock.Anything, int64(1), mock.AnythingOfType("*model.UpdatePostDTO")).Return(&model.Post{}, nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return([]*model.PostMedia{{ID: 10}}, nil)
				mediaRepo.On("Detach", mock.Anything, []int64{10}).Return(nil)
				mediaRepo.On("Attach", mock.Anything, int64(1), mock.AnythingOfType("[]*model.PostMedia")).Return(errors.New("attach error"))
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				postID: 1,
				post: &model.UpdatePostDTO{
					MediaItems: []*model.PostMediaInput{{URL: "new_url", Type: "image", Position: 1}},
				},
			},
			wantErr:     true,
			wantErrType: custom_errors.ErrMediaAttachFailed,
		},
		{
			name: "Error creating tag",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo) // For defer
				tx.On("TagRepository").Return(tagRepo)
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 1}, nil)
				postRepo.On("Update", mock.Anything, int64(1), mock.AnythingOfType("*model.UpdatePostDTO")).Return(&model.Post{}, nil)
				// No media items for this test case
				tagRepo.On("Create", mock.Anything, "newtag").Return(nil, custom_errors.ErrTagCreateFailed)
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				postID: 1,
				post: &model.UpdatePostDTO{
					Tags: []string{"newtag"},
				},
			},
			wantErr:     true,
			wantErrType: custom_errors.ErrTagCreateFailed,
		},
		{
			name: "Error replacing post tags",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo) // For defer
				tx.On("TagRepository").Return(tagRepo)
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 1}, nil)
				postRepo.On("Update", mock.Anything, int64(1), mock.AnythingOfType("*model.UpdatePostDTO")).Return(&model.Post{}, nil)
				tagRepo.On("Create", mock.Anything, "newtag").Return(&model.Tag{ID: 1, Name: "newtag"}, nil) // Or ErrTagAlreadyExists
				tagRepo.On("ReplacePostTags", mock.Anything, int64(1), []string{"newtag"}).Return(custom_errors.ErrTagPost)
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				postID: 1,
				post: &model.UpdatePostDTO{
					Tags: []string{"newtag"},
				},
			},
			wantErr:     true,
			wantErrType: custom_errors.ErrTagPost,
		},
		{
			name: "Error committing transaction",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)
				tx.On("TagRepository").Return(tagRepo)
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 1}, nil)
				postRepo.On("Update", mock.Anything, int64(1), mock.AnythingOfType("*model.UpdatePostDTO")).Return(&model.Post{}, nil)
				// No media or tags for simplicity
				tx.On("Commit", mock.Anything).Return(errors.New("commit error"))
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				postID: 1,
				post:   &model.UpdatePostDTO{Title: func() *string { s := "t"; return &s }()},
			},
			wantErr:     true,
			wantErrType: custom_errors.ErrDatabaseQuery,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postRepo := new(post_repository_mock.Repository)
			tagRepo := new(tag_repository_mock.Repository)
			mediaRepo := new(media_repository_mock.Repository)
			uow := new(postgres_mock.UnitOfWork)
			userClient := new(user_client_mock.Client) // Not used in UpdatePost but part of service
			tx := new(postgres_mock.Transaction)
			metrics := prometheus.NewPrometheusMetricsProvider()

			if tt.mocks != nil {
				tt.mocks(postRepo, tagRepo, mediaRepo, uow, tx)
			}

			s := NewPostService(postRepo, tagRepo, mediaRepo, uow, log, userClient, metrics)
			err := s.UpdatePost(tt.args.ctx, tt.args.userID, tt.args.postID, tt.args.post)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrType != nil {
					assert.True(t, errors.Is(err, tt.wantErrType), "expected error type %T, got %T", tt.wantErrType, err)
				}
			} else {
				assert.NoError(t, err)
			}

			postRepo.AssertExpectations(t)
			tagRepo.AssertExpectations(t)
			mediaRepo.AssertExpectations(t)
			uow.AssertExpectations(t)
			tx.AssertExpectations(t)
		})
	}
}

func TestPostService_DeletePost(t *testing.T) {
	log := logger.New("test")
	type args struct {
		ctx    context.Context
		userID int64
		postID int64
	}
	tests := []struct {
		name        string
		mocks       func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction)
		args        args
		wantErr     bool
		wantErrType error
	}{
		{
			name: "Success",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)
				tx.On("TagRepository").Return(tagRepo)

				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 1}, nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return([]*model.PostMedia{{ID: 10}}, nil)
				mediaRepo.On("Detach", mock.Anything, []int64{10}).Return(nil)
				tagRepo.On("FindByPost", mock.Anything, int64(1)).Return([]*model.Tag{{ID: 1, Name: "tag1"}}, nil)
				tagRepo.On("UntagPost", mock.Anything, int64(1), []string{"tag1"}).Return(nil)
				postRepo.On("Delete", mock.Anything, int64(1)).Return(nil)
				tx.On("Commit", mock.Anything).Return(nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				postID: 1,
			},
			wantErr: false,
		},
		{
			name: "Success with no media",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)
				tx.On("TagRepository").Return(tagRepo)

				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 1}, nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return(nil, custom_errors.ErrMediaNotFound) // No media
				// Detach should not be called
				tagRepo.On("FindByPost", mock.Anything, int64(1)).Return([]*model.Tag{{ID: 1, Name: "tag1"}}, nil)
				tagRepo.On("UntagPost", mock.Anything, int64(1), []string{"tag1"}).Return(nil)
				postRepo.On("Delete", mock.Anything, int64(1)).Return(nil)
				tx.On("Commit", mock.Anything).Return(nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				postID: 1,
			},
			wantErr: false,
		},
		{
			name: "Success with no tags",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)
				tx.On("TagRepository").Return(tagRepo)

				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 1}, nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return([]*model.PostMedia{{ID: 10}}, nil)
				mediaRepo.On("Detach", mock.Anything, []int64{10}).Return(nil)
				tagRepo.On("FindByPost", mock.Anything, int64(1)).Return(nil, custom_errors.ErrTagsNotFound) // No tags
				// UntagPost should not be called
				postRepo.On("Delete", mock.Anything, int64(1)).Return(nil)
				tx.On("Commit", mock.Anything).Return(nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				postID: 1,
			},
			wantErr: false,
		},
		{
			name: "Error begin transaction",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(nil, errors.New("db error"))
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				postID: 1,
			},
			wantErr:     true,
			wantErrType: custom_errors.ErrDatabaseQuery,
		},
		{
			name: "Error GetByID post not found",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo) // For defer
				tx.On("TagRepository").Return(tagRepo)     // For defer
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(nil, custom_errors.ErrPostNotFound)
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				postID: 1,
			},
			wantErr:     true,
			wantErrType: custom_errors.ErrPostNotFound,
		},
		{
			name: "Error user is not author",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 2}, nil) // Different AuthorID
				// Важно: эти методы должны быть доступны до их вызова в defer
				tx.On("MediaRepository").Return(mediaRepo) // Для defer
				tx.On("TagRepository").Return(tagRepo)     // Для defer
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 1, // UserID is 1
				postID: 1,
			},
			wantErr:     true,
			wantErrType: custom_errors.ErrForbidden,
		},
		{
			name: "Error getting media for post (not ErrMediaNotFound)",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)
				tx.On("TagRepository").Return(tagRepo) // For defer
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 1}, nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return(nil, errors.New("db error"))
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				postID: 1,
			},
			wantErr:     true,
			wantErrType: custom_errors.ErrMediaQueryFailed,
		},
		{
			name: "Error detaching media (not ErrMediaNotFound)",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)
				tx.On("TagRepository").Return(tagRepo) // For defer
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 1}, nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return([]*model.PostMedia{{ID: 10}}, nil)
				mediaRepo.On("Detach", mock.Anything, []int64{10}).Return(errors.New("detach error"))
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				postID: 1,
			},
			wantErr:     true,
			wantErrType: custom_errors.ErrMediaDetachFailed,
		},
		{
			name: "Error finding tags for post (not ErrTagsNotFound)",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)
				tx.On("TagRepository").Return(tagRepo)
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 1}, nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return(nil, custom_errors.ErrMediaNotFound) // No media, proceed
				tagRepo.On("FindByPost", mock.Anything, int64(1)).Return(nil, errors.New("db error"))
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				postID: 1,
			},
			wantErr:     true,
			wantErrType: custom_errors.ErrTagQueryFailed,
		},
		{
			name: "Error untagging post (not ErrTagNotFound)",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)
				tx.On("TagRepository").Return(tagRepo)
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 1}, nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return(nil, custom_errors.ErrMediaNotFound) // No media
				tagRepo.On("FindByPost", mock.Anything, int64(1)).Return([]*model.Tag{{ID: 1, Name: "tag1"}}, nil)
				tagRepo.On("UntagPost", mock.Anything, int64(1), []string{"tag1"}).Return(errors.New("untag error"))
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				postID: 1,
			},
			wantErr:     true,
			wantErrType: custom_errors.ErrTagDeleteFailed,
		},
		{
			name: "Error deleting post from repo",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)
				tx.On("TagRepository").Return(tagRepo)
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 1}, nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return(nil, custom_errors.ErrMediaNotFound)
				tagRepo.On("FindByPost", mock.Anything, int64(1)).Return(nil, custom_errors.ErrTagsNotFound)
				postRepo.On("Delete", mock.Anything, int64(1)).Return(custom_errors.ErrDatabaseQuery)
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				postID: 1,
			},
			wantErr:     true,
			wantErrType: custom_errors.ErrDatabaseQuery,
		},
		{
			name: "Error committing transaction",
			mocks: func(postRepo *post_repository_mock.Repository, tagRepo *tag_repository_mock.Repository, mediaRepo *media_repository_mock.Repository, uow *postgres_mock.UnitOfWork, tx *postgres_mock.Transaction) {
				uow.On("Begin", mock.Anything).Return(tx, nil)
				tx.On("PostRepository").Return(postRepo)
				tx.On("MediaRepository").Return(mediaRepo)
				tx.On("TagRepository").Return(tagRepo)
				postRepo.On("GetByID", mock.Anything, int64(1)).Return(&model.Post{ID: 1, AuthorID: 1}, nil)
				mediaRepo.On("GetByPost", mock.Anything, int64(1)).Return(nil, custom_errors.ErrMediaNotFound)
				tagRepo.On("FindByPost", mock.Anything, int64(1)).Return(nil, custom_errors.ErrTagsNotFound)
				postRepo.On("Delete", mock.Anything, int64(1)).Return(nil)
				tx.On("Commit", mock.Anything).Return(errors.New("commit error"))
				tx.On("Rollback", mock.Anything).Return(nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				postID: 1,
			},
			wantErr:     true,
			wantErrType: custom_errors.ErrDatabaseQuery,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postRepo := new(post_repository_mock.Repository)
			tagRepo := new(tag_repository_mock.Repository)
			mediaRepo := new(media_repository_mock.Repository)
			uow := new(postgres_mock.UnitOfWork)
			userClient := new(user_client_mock.Client) // Not used in DeletePost but part of service
			tx := new(postgres_mock.Transaction)
			metrics := prometheus.NewPrometheusMetricsProvider()

			if tt.mocks != nil {
				tt.mocks(postRepo, tagRepo, mediaRepo, uow, tx)
			}

			s := NewPostService(postRepo, tagRepo, mediaRepo, uow, log, userClient, metrics)
			err := s.DeletePost(tt.args.ctx, tt.args.userID, tt.args.postID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrType != nil {
					assert.True(t, errors.Is(err, tt.wantErrType), "expected error type %T, got %T", tt.wantErrType, err)
				}
			} else {
				assert.NoError(t, err)
			}

			postRepo.AssertExpectations(t)
			tagRepo.AssertExpectations(t)
			mediaRepo.AssertExpectations(t)
			uow.AssertExpectations(t)
			tx.AssertExpectations(t)
		})
	}
}
