package post_grpc_test

import (
	"context"
	"errors"
	"github.com/soloda1/pinstack-proto-definitions/custom_errors"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
	pb "github.com/soloda1/pinstack-proto-definitions/gen/go/pinstack-proto-definitions/post/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	post_grpc "pinstack-post-service/internal/delivery/grpc/post"
	"pinstack-post-service/internal/logger"
	"pinstack-post-service/internal/model"
	mockpost "pinstack-post-service/mocks/post"
)

func TestCreatePostHandler_CreatePost(t *testing.T) {
	validate := validator.New()
	testLogger := logger.New("test")

	t.Run("Success", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewCreatePostHandler(mockPostService, validate, testLogger)

		req := &pb.CreatePostRequest{
			AuthorId: 123,
			Title:    "Test Post Title",
			Content:  "This is a test post content with enough length",
			Tags:     []string{"tag1", "tag2"},
			Media: []*pb.MediaInput{
				{
					Url:      "https://example.com/image.jpg",
					Type:     "image",
					Position: 1,
				},
			},
		}

		createdAt := time.Now()
		updatedAt := time.Now()

		expectedPostDetailed := &model.PostDetailed{
			Post: &model.Post{
				ID:        1,
				AuthorID:  123,
				Title:     "Test Post Title",
				Content:   &req.Content,
				CreatedAt: pgtype.Timestamp{Time: createdAt, Valid: true},
				UpdatedAt: pgtype.Timestamp{Time: updatedAt, Valid: true},
			},
			Media: []*model.PostMedia{
				{
					ID:        1,
					PostID:    1,
					URL:       "https://example.com/image.jpg",
					Type:      "image",
					Position:  1,
					CreatedAt: pgtype.Timestamptz{Time: createdAt, Valid: true},
				},
			},
			Tags: []*model.Tag{
				{ID: 1, Name: "tag1"},
				{ID: 2, Name: "tag2"},
			},
		}

		mockPostService.On("CreatePost", mock.Anything, mock.MatchedBy(func(dto *model.CreatePostDTO) bool {
			return dto.AuthorID == req.AuthorId &&
				dto.Title == req.Title &&
				*dto.Content == req.Content &&
				len(dto.Tags) == len(req.Tags) &&
				len(dto.MediaItems) == len(req.Media)
		})).Return(expectedPostDetailed, nil)

		resp, err := handler.CreatePost(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, expectedPostDetailed.Post.ID, resp.Id)
		assert.Equal(t, expectedPostDetailed.Post.AuthorID, resp.AuthorId)
		assert.Equal(t, expectedPostDetailed.Post.Title, resp.Title)
		assert.Equal(t, *expectedPostDetailed.Post.Content, resp.Content)
		assert.Equal(t, len(expectedPostDetailed.Tags), len(resp.Tags))
		assert.Equal(t, len(expectedPostDetailed.Media), len(resp.Media))
		mockPostService.AssertExpectations(t)
	})

	t.Run("ValidationError", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewCreatePostHandler(mockPostService, validate, testLogger)

		req := &pb.CreatePostRequest{
			AuthorId: 123,
			Title:    "A",
			Content:  "Short",
		}

		resp, err := handler.CreatePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())
		assert.Equal(t, "invalid request", statusErr.Message())

		mockPostService.AssertNotCalled(t, "CreatePost")
	})

	t.Run("ServiceValidationError", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewCreatePostHandler(mockPostService, validate, testLogger)

		req := &pb.CreatePostRequest{
			AuthorId: 123,
			Title:    "Test Post Title",
			Content:  "This is a test post content with enough length",
		}

		mockPostService.On("CreatePost", mock.Anything, mock.Anything).
			Return(nil, custom_errors.ErrPostValidation)

		resp, err := handler.CreatePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())
		mockPostService.AssertExpectations(t)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewCreatePostHandler(mockPostService, validate, testLogger)

		req := &pb.CreatePostRequest{
			AuthorId: 123,
			Title:    "Test Post Title",
			Content:  "This is a test post content with enough length",
		}

		mockPostService.On("CreatePost", mock.Anything, mock.Anything).
			Return(nil, errors.New("database error"))

		resp, err := handler.CreatePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, statusErr.Code())
		mockPostService.AssertExpectations(t)
	})

	t.Run("MediaTypeValidation", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewCreatePostHandler(mockPostService, validate, testLogger)

		req := &pb.CreatePostRequest{
			AuthorId: 123,
			Title:    "Test Post Title",
			Content:  "This is a test post content with enough length",
			Media: []*pb.MediaInput{
				{
					Url:      "invalid-url",
					Type:     "image",
					Position: 1,
				},
			},
		}

		resp, err := handler.CreatePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())

		mockPostService.AssertNotCalled(t, "CreatePost")
	})

	t.Run("SuccessWithNullableFields", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewCreatePostHandler(mockPostService, validate, testLogger)

		req := &pb.CreatePostRequest{
			AuthorId: 123,
			Title:    "Test Post Title",
			Content:  "This is a test post content with enough length",
		}

		expectedPostDetailed := &model.PostDetailed{
			Post: &model.Post{
				ID:        1,
				AuthorID:  123,
				Title:     "Test Post Title",
				Content:   &req.Content,
				CreatedAt: pgtype.Timestamp{Valid: false},
				UpdatedAt: pgtype.Timestamp{Valid: false},
			},
			Media: nil,
			Tags:  nil,
		}

		mockPostService.On("CreatePost", mock.Anything, mock.Anything).
			Return(expectedPostDetailed, nil)

		resp, err := handler.CreatePost(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, expectedPostDetailed.Post.ID, resp.Id)
		assert.Equal(t, expectedPostDetailed.Post.AuthorID, resp.AuthorId)
		assert.Nil(t, resp.CreatedAt)
		assert.Nil(t, resp.UpdatedAt)
		assert.Empty(t, resp.Media)
		assert.Empty(t, resp.Tags)
		mockPostService.AssertExpectations(t)
	})

	t.Run("CompleteDataFlow", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewCreatePostHandler(mockPostService, validate, testLogger)

		createdAt := time.Now()
		updatedAt := time.Now()

		req := &pb.CreatePostRequest{
			AuthorId: 123,
			Title:    "Complete Test",
			Content:  "This is a complete test with media and tags",
			Tags:     []string{"tech", "golang", "testing"},
			Media: []*pb.MediaInput{
				{
					Url:      "https://example.com/image1.jpg",
					Type:     "image",
					Position: 1,
				},
				{
					Url:      "https://example.com/image2.jpg",
					Type:     "image",
					Position: 2,
				},
			},
		}

		expectedPostDetailed := &model.PostDetailed{
			Post: &model.Post{
				ID:        42,
				AuthorID:  123,
				Title:     "Complete Test",
				Content:   &req.Content,
				CreatedAt: pgtype.Timestamp{Time: createdAt, Valid: true},
				UpdatedAt: pgtype.Timestamp{Time: updatedAt, Valid: true},
			},
			Media: []*model.PostMedia{
				{
					ID:        1,
					PostID:    42,
					URL:       "https://example.com/image1.jpg",
					Type:      "image",
					Position:  1,
					CreatedAt: pgtype.Timestamptz{Time: createdAt, Valid: true},
				},
				{
					ID:        2,
					PostID:    42,
					URL:       "https://example.com/image2.jpg",
					Type:      "image",
					Position:  2,
					CreatedAt: pgtype.Timestamptz{Time: createdAt, Valid: true},
				},
			},
			Tags: []*model.Tag{
				{ID: 1, Name: "tech"},
				{ID: 2, Name: "golang"},
				{ID: 3, Name: "testing"},
			},
		}

		mockPostService.On("CreatePost", mock.Anything, mock.MatchedBy(func(dto *model.CreatePostDTO) bool {
			if dto.AuthorID != req.AuthorId || dto.Title != req.Title || *dto.Content != req.Content {
				return false
			}

			if len(dto.Tags) != len(req.Tags) {
				return false
			}

			for i, tag := range dto.Tags {
				if tag != req.Tags[i] {
					return false
				}
			}

			if len(dto.MediaItems) != len(req.Media) {
				return false
			}

			for i, media := range dto.MediaItems {
				if media.URL != req.Media[i].Url ||
					string(media.Type) != req.Media[i].Type ||
					media.Position != req.Media[i].Position {
					return false
				}
			}

			return true
		})).Return(expectedPostDetailed, nil)

		resp, err := handler.CreatePost(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp)

		assert.Equal(t, expectedPostDetailed.Post.ID, resp.Id)
		assert.Equal(t, expectedPostDetailed.Post.AuthorID, resp.AuthorId)
		assert.Equal(t, expectedPostDetailed.Post.Title, resp.Title)
		assert.Equal(t, *expectedPostDetailed.Post.Content, resp.Content)

		assert.NotNil(t, resp.CreatedAt)
		assert.Equal(t, timestamppb.New(createdAt).Seconds, resp.CreatedAt.Seconds)
		assert.NotNil(t, resp.UpdatedAt)
		assert.Equal(t, timestamppb.New(updatedAt).Seconds, resp.UpdatedAt.Seconds)

		assert.Equal(t, len(expectedPostDetailed.Tags), len(resp.Tags))
		for i, tag := range expectedPostDetailed.Tags {
			assert.Equal(t, tag.Name, resp.Tags[i])
		}

		assert.Equal(t, len(expectedPostDetailed.Media), len(resp.Media))
		for i, media := range expectedPostDetailed.Media {
			assert.Equal(t, media.ID, resp.Media[i].Id)
			assert.Equal(t, media.URL, resp.Media[i].Url)
			assert.Equal(t, string(media.Type), resp.Media[i].Type)
			assert.Equal(t, media.Position, resp.Media[i].Position)
		}

		mockPostService.AssertExpectations(t)
	})
}
