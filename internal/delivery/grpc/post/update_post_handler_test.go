package post_grpc_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	pb "github.com/soloda1/pinstack-proto-definitions/gen/go/pinstack-proto-definitions/post/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/jackc/pgx/v5/pgtype"
	"pinstack-post-service/internal/custom_errors"
	post_grpc "pinstack-post-service/internal/delivery/grpc/post"
	"pinstack-post-service/internal/model"
	mockpost "pinstack-post-service/mocks/post"
)

func TestUpdatePostHandler_UpdatePost(t *testing.T) {
	validate := validator.New()

	t.Run("Success_FullUpdate", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewUpdatePostHandler(mockPostService, validate)

		userID := int64(123)
		postID := int64(456)
		title := "Updated Title"
		content := "This is the updated content for the post with sufficient length"

		req := &pb.UpdatePostRequest{
			UserId:  userID,
			Id:      postID,
			Title:   title,
			Content: content,
			Tags:    []string{"updated", "golang"},
			Media: []*pb.MediaInput{
				{
					Url:      "https://example.com/new-image.jpg",
					Type:     "image",
					Position: 1,
				},
			},
		}

		mockPostService.On("UpdatePost", mock.Anything, userID, postID, mock.MatchedBy(func(dto *model.UpdatePostDTO) bool {
			return dto.UserID == req.GetUserId() &&
				*dto.Title == req.Title &&
				*dto.Content == req.Content &&
				len(dto.Tags) == len(req.Tags) &&
				len(dto.MediaItems) == len(req.Media)
		})).Return(nil)

		createdAt := time.Now().Add(-24 * time.Hour)
		updatedAt := time.Now()

		expectedPostDetailed := &model.PostDetailed{
			Post: &model.Post{
				ID:        postID,
				AuthorID:  userID,
				Title:     title,
				Content:   &content,
				CreatedAt: pgtype.Timestamp{Time: createdAt, Valid: true},
				UpdatedAt: pgtype.Timestamp{Time: updatedAt, Valid: true},
			},
			Media: []*model.PostMedia{
				{
					ID:        1,
					PostID:    postID,
					URL:       "https://example.com/new-image.jpg",
					Type:      "image",
					Position:  1,
					CreatedAt: pgtype.Timestamptz{Time: updatedAt, Valid: true},
				},
			},
			Tags: []*model.Tag{
				{ID: 1, Name: "updated"},
				{ID: 2, Name: "golang"},
			},
		}

		mockPostService.On("GetPostByID", mock.Anything, postID).Return(expectedPostDetailed, nil)

		resp, err := handler.UpdatePost(context.Background(), req)

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

	t.Run("Success_PartialUpdate", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewUpdatePostHandler(mockPostService, validate)

		userID := int64(123)
		postID := int64(456)
		title := "Only Title Updated"

		req := &pb.UpdatePostRequest{
			UserId:  userID,
			Id:      postID,
			Title:   title,
			Content: "",
			Tags:    []string{},
		}

		mockPostService.On("UpdatePost", mock.Anything, userID, postID, mock.MatchedBy(func(dto *model.UpdatePostDTO) bool {
			return dto.UserID == req.GetUserId() &&
				*dto.Title == req.Title &&
				*dto.Content == req.Content &&
				len(dto.Tags) == 0 &&
				len(dto.MediaItems) == 0
		})).Return(nil)

		createdAt := time.Now().Add(-24 * time.Hour)
		updatedAt := time.Now()
		originalContent := "Original content that wasn't changed"

		expectedPostDetailed := &model.PostDetailed{
			Post: &model.Post{
				ID:        postID,
				AuthorID:  userID,
				Title:     title,
				Content:   &originalContent,
				CreatedAt: pgtype.Timestamp{Time: createdAt, Valid: true},
				UpdatedAt: pgtype.Timestamp{Time: updatedAt, Valid: true},
			},
			Media: nil,
			Tags:  nil,
		}

		mockPostService.On("GetPostByID", mock.Anything, postID).Return(expectedPostDetailed, nil)

		resp, err := handler.UpdatePost(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, expectedPostDetailed.Post.ID, resp.Id)
		assert.Equal(t, title, resp.Title)
		assert.Equal(t, originalContent, resp.Content)

		mockPostService.AssertExpectations(t)
	})

	t.Run("ValidationError", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewUpdatePostHandler(mockPostService, validate)

		req := &pb.UpdatePostRequest{
			UserId:  123,
			Id:      0,
			Title:   "Test Title",
			Content: "Test Content",
		}

		resp, err := handler.UpdatePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())
		assert.Contains(t, statusErr.Message(), "invalid request")

		mockPostService.AssertNotCalled(t, "UpdatePost")
		mockPostService.AssertNotCalled(t, "GetPostByID")
	})

	t.Run("ValidationError_InvalidMedia", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewUpdatePostHandler(mockPostService, validate)

		req := &pb.UpdatePostRequest{
			UserId:  123,
			Id:      456,
			Title:   "Test Title",
			Content: "Test Content",
			Media: []*pb.MediaInput{
				{
					Url:      "invalid-url",
					Type:     "image",
					Position: 1,
				},
			},
		}

		resp, err := handler.UpdatePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())

		mockPostService.AssertNotCalled(t, "UpdatePost")
	})

	t.Run("PostNotFound", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewUpdatePostHandler(mockPostService, validate)

		userID := int64(123)
		postID := int64(999)
		title := "Updated Title"
		content := "Updated Content"

		req := &pb.UpdatePostRequest{
			UserId:  userID,
			Id:      postID,
			Title:   title,
			Content: content,
		}

		mockPostService.On("UpdatePost", mock.Anything, userID, postID, mock.Anything).
			Return(custom_errors.ErrPostNotFound)

		resp, err := handler.UpdatePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, statusErr.Code())
		assert.Contains(t, statusErr.Message(), "post not found")

		mockPostService.AssertNotCalled(t, "GetPostByID")
	})

	t.Run("ValidationErrorFromService", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewUpdatePostHandler(mockPostService, validate)

		userID := int64(123)
		postID := int64(456)
		title := "Updated Title"
		content := "Updated Content"

		req := &pb.UpdatePostRequest{
			UserId:  userID,
			Id:      postID,
			Title:   title,
			Content: content,
		}

		mockPostService.On("UpdatePost", mock.Anything, userID, postID, mock.Anything).
			Return(custom_errors.ErrPostValidation)

		resp, err := handler.UpdatePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())
		assert.Contains(t, statusErr.Message(), "validation failed")
	})

	t.Run("NotAuthorError", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewUpdatePostHandler(mockPostService, validate)

		userID := int64(123)
		postID := int64(456)
		title := "Updated Title"
		content := "Updated Content"

		req := &pb.UpdatePostRequest{
			UserId:  userID,
			Id:      postID,
			Title:   title,
			Content: content,
		}

		mockPostService.On("UpdatePost", mock.Anything, userID, postID, mock.Anything).
			Return(custom_errors.ErrInvalidInput)

		resp, err := handler.UpdatePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, statusErr.Code())
		assert.Contains(t, statusErr.Message(), "user is not author")
	})

	t.Run("InternalError_Update", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewUpdatePostHandler(mockPostService, validate)

		userID := int64(123)
		postID := int64(456)
		title := "Updated Title"
		content := "Updated Content"

		req := &pb.UpdatePostRequest{
			UserId:  userID,
			Id:      postID,
			Title:   title,
			Content: content,
		}

		mockPostService.On("UpdatePost", mock.Anything, userID, postID, mock.Anything).
			Return(errors.New("database error"))

		resp, err := handler.UpdatePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, statusErr.Code())
		assert.Contains(t, statusErr.Message(), "failed to update post")
	})

	t.Run("GetError_AfterSuccessfulUpdate", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewUpdatePostHandler(mockPostService, validate)

		userID := int64(123)
		postID := int64(456)
		title := "Updated Title"
		content := "Updated Content"

		req := &pb.UpdatePostRequest{
			UserId:  userID,
			Id:      postID,
			Title:   title,
			Content: content,
		}

		mockPostService.On("UpdatePost", mock.Anything, userID, postID, mock.Anything).
			Return(nil)

		mockPostService.On("GetPostByID", mock.Anything, postID).
			Return(nil, custom_errors.ErrPostNotFound)

		resp, err := handler.UpdatePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, statusErr.Code())
		assert.Contains(t, statusErr.Message(), "post not found")
	})

	t.Run("InternalError_Get", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewUpdatePostHandler(mockPostService, validate)

		userID := int64(123)
		postID := int64(456)
		title := "Updated Title"
		content := "Updated Content"

		req := &pb.UpdatePostRequest{
			UserId:  userID,
			Id:      postID,
			Title:   title,
			Content: content,
		}

		mockPostService.On("UpdatePost", mock.Anything, userID, postID, mock.Anything).
			Return(nil)

		mockPostService.On("GetPostByID", mock.Anything, postID).
			Return(nil, errors.New("database error"))

		resp, err := handler.UpdatePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, statusErr.Code())
		assert.Contains(t, statusErr.Message(), "failed to update post")
	})
}
