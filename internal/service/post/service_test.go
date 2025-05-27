package post_service

import (
	"context"
	"errors"
	"testing"

	user_client_mock "pinstack-post-service/mocks/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"pinstack-post-service/internal/custom_errors"
	"pinstack-post-service/internal/logger"
	"pinstack-post-service/internal/model"
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

			if tt.mocks != nil {
				tt.mocks(postRepo, tagRepo, mediaRepo, uow, userClient, tx)
			}

			s := NewPostService(postRepo, tagRepo, mediaRepo, uow, log, userClient)
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

			if tt.mocks != nil {
				tt.mocks(postRepo, mediaRepo, tagRepo, userClient)
			}

			s := NewPostService(postRepo, tagRepo, mediaRepo, uow, log, userClient)
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
