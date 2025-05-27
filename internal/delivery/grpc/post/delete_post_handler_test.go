package post_grpc_test

import (
	"context"
	"errors"
	"github.com/go-playground/validator/v10"
	pb "github.com/soloda1/pinstack-proto-definitions/gen/go/pinstack-proto-definitions/post/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"pinstack-post-service/internal/custom_errors"
	post_grpc "pinstack-post-service/internal/delivery/grpc/post"
	mockpost "pinstack-post-service/mocks/post"
	"testing"
)

func TestDeletePostHandler_DeletePost(t *testing.T) {
	validate := validator.New()

	t.Run("Success", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewDeletePostHandler(mockPostService, validate)

		req := &pb.DeletePostRequest{
			UserId: 123,
			Id:     456,
		}

		mockPostService.On("DeletePost", mock.Anything, int64(123), int64(456)).Return(nil)

		resp, err := handler.DeletePost(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		mockPostService.AssertExpectations(t)
	})

	t.Run("ValidationError", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewDeletePostHandler(mockPostService, validate)

		req := &pb.DeletePostRequest{
			UserId: 123,
			Id:     0,
		}

		resp, err := handler.DeletePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())
		assert.Contains(t, statusErr.Message(), "invalid request")

		mockPostService.AssertNotCalled(t, "DeletePost")
	})

	t.Run("PostNotFound", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewDeletePostHandler(mockPostService, validate)

		req := &pb.DeletePostRequest{
			UserId: 123,
			Id:     456,
		}

		mockPostService.On("DeletePost", mock.Anything, int64(123), int64(456)).
			Return(custom_errors.ErrPostNotFound)

		resp, err := handler.DeletePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, statusErr.Code())
		assert.Contains(t, statusErr.Message(), "not found")
		mockPostService.AssertExpectations(t)
	})

	t.Run("ValidationError_FromService", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewDeletePostHandler(mockPostService, validate)

		req := &pb.DeletePostRequest{
			UserId: 123,
			Id:     456,
		}

		mockPostService.On("DeletePost", mock.Anything, int64(123), int64(456)).
			Return(custom_errors.ErrPostValidation)

		resp, err := handler.DeletePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())
		assert.Contains(t, statusErr.Message(), "validation failed")
		mockPostService.AssertExpectations(t)
	})

	t.Run("InternalError", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewDeletePostHandler(mockPostService, validate)

		req := &pb.DeletePostRequest{
			UserId: 123,
			Id:     456,
		}

		mockPostService.On("DeletePost", mock.Anything, int64(123), int64(456)).
			Return(errors.New("database error"))

		resp, err := handler.DeletePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, statusErr.Code())
		assert.Contains(t, statusErr.Message(), "failed to delete post")
		mockPostService.AssertExpectations(t)
	})
}
