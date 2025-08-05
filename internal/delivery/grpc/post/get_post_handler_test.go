package post_grpc_test

import (
	"context"
	"errors"
	"github.com/soloda1/pinstack-proto-definitions/custom_errors"
	post_grpc "pinstack-post-service/internal/delivery/grpc/post"
	"pinstack-post-service/internal/logger"
	"pinstack-post-service/internal/model"
	mockpost "pinstack-post-service/mocks/post"
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
)

func TestGetPostHandler_GetPost(t *testing.T) {
	validate := validator.New()
	testLogger := logger.New("test")

	t.Run("Success", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewGetPostHandler(mockPostService, validate, testLogger)

		postID := int64(123)
		req := &pb.GetPostRequest{
			Id: postID,
		}

		createdAt := time.Now()
		updatedAt := time.Now()
		content := "This is the post content"

		expectedPostDetailed := &model.PostDetailed{
			Post: &model.Post{
				ID:        postID,
				AuthorID:  456,
				Title:     "Test Post Title",
				Content:   &content,
				CreatedAt: pgtype.Timestamp{Time: createdAt, Valid: true},
				UpdatedAt: pgtype.Timestamp{Time: updatedAt, Valid: true},
			},
			Media: []*model.PostMedia{
				{
					ID:        1,
					PostID:    postID,
					URL:       "https://example.com/image1.jpg",
					Type:      "image",
					Position:  1,
					CreatedAt: pgtype.Timestamptz{Time: createdAt, Valid: true},
				},
				{
					ID:        2,
					PostID:    postID,
					URL:       "https://example.com/image2.jpg",
					Type:      "video",
					Position:  1,
					CreatedAt: pgtype.Timestamptz{Time: createdAt, Valid: true},
				},
			},
			Tags: []*model.Tag{
				{ID: 1, Name: "tag1"},
				{ID: 2, Name: "tag2"},
			},
		}

		mockPostService.On("GetPostByID", mock.Anything, postID).Return(expectedPostDetailed, nil)

		resp, err := handler.GetPost(context.Background(), req)

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

	t.Run("SuccessWithNullableFields", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewGetPostHandler(mockPostService, validate, testLogger)

		postID := int64(123)
		req := &pb.GetPostRequest{
			Id: postID,
		}

		expectedPostDetailed := &model.PostDetailed{
			Post: &model.Post{
				ID:        postID,
				AuthorID:  456,
				Title:     "Test Post Title",
				Content:   nil,
				CreatedAt: pgtype.Timestamp{Valid: false},
				UpdatedAt: pgtype.Timestamp{Valid: false},
			},
			Media: nil,
			Tags:  nil,
		}

		mockPostService.On("GetPostByID", mock.Anything, postID).Return(expectedPostDetailed, nil)

		resp, err := handler.GetPost(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, expectedPostDetailed.Post.ID, resp.Id)
		assert.Equal(t, expectedPostDetailed.Post.AuthorID, resp.AuthorId)
		assert.Equal(t, expectedPostDetailed.Post.Title, resp.Title)
		assert.Empty(t, resp.Content)
		assert.Nil(t, resp.CreatedAt)
		assert.Nil(t, resp.UpdatedAt)
		assert.Empty(t, resp.Media)
		assert.Empty(t, resp.Tags)

		mockPostService.AssertExpectations(t)
	})

	t.Run("ValidationError", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewGetPostHandler(mockPostService, validate, testLogger)

		req := &pb.GetPostRequest{
			Id: 0,
		}

		resp, err := handler.GetPost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())
		assert.Equal(t, "invalid request", statusErr.Message())

		mockPostService.AssertNotCalled(t, "GetPostByID")
	})

	t.Run("PostNotFound", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewGetPostHandler(mockPostService, validate, testLogger)

		postID := int64(999)
		req := &pb.GetPostRequest{
			Id: postID,
		}

		mockPostService.On("GetPostByID", mock.Anything, postID).Return(nil, custom_errors.ErrPostNotFound)

		resp, err := handler.GetPost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, statusErr.Code())
		assert.Equal(t, "post not found", statusErr.Message())

		mockPostService.AssertExpectations(t)
	})

	t.Run("ValidationErrorFromService", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewGetPostHandler(mockPostService, validate, testLogger)

		postID := int64(123)
		req := &pb.GetPostRequest{
			Id: postID,
		}

		mockPostService.On("GetPostByID", mock.Anything, postID).Return(nil, custom_errors.ErrPostValidation)

		resp, err := handler.GetPost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())
		assert.Equal(t, "post retrieval validation failed", statusErr.Message())

		mockPostService.AssertExpectations(t)
	})

	t.Run("InternalError", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewGetPostHandler(mockPostService, validate, testLogger)

		postID := int64(123)
		req := &pb.GetPostRequest{
			Id: postID,
		}

		mockPostService.On("GetPostByID", mock.Anything, postID).Return(nil, errors.New("database error"))

		resp, err := handler.GetPost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, statusErr.Code())
		assert.Equal(t, "failed to get post", statusErr.Message())

		mockPostService.AssertExpectations(t)
	})
}
