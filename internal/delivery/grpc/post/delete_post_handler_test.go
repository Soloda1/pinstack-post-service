package post_grpc_test

import (
	"context"
	"errors"
	"github.com/soloda1/pinstack-proto-definitions/custom_errors"
	"testing"

	"github.com/go-playground/validator/v10"
	pb "github.com/soloda1/pinstack-proto-definitions/gen/go/pinstack-proto-definitions/post/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	post_grpc "pinstack-post-service/internal/delivery/grpc/post"
	"pinstack-post-service/internal/logger"
	mockpost "pinstack-post-service/mocks/post"
)

func TestDeletePostHandler_DeletePost(t *testing.T) {
	validate := validator.New()
	testLogger := logger.New("test")

	t.Run("Success", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewDeletePostHandler(mockPostService, validate, testLogger)

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
		handler := post_grpc.NewDeletePostHandler(mockPostService, validate, testLogger)

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
		assert.Equal(t, "invalid request", statusErr.Message())

		mockPostService.AssertNotCalled(t, "DeletePost")
	})

	t.Run("PostNotFound", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewDeletePostHandler(mockPostService, validate, testLogger)

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
		assert.Equal(t, "post not found", statusErr.Message())
		mockPostService.AssertExpectations(t)
	})

	t.Run("ValidationError_FromService", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewDeletePostHandler(mockPostService, validate, testLogger)

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
		assert.Equal(t, "validation failed", statusErr.Message())
		mockPostService.AssertExpectations(t)
	})

	t.Run("ForbiddenError", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewDeletePostHandler(mockPostService, validate, testLogger)

		req := &pb.DeletePostRequest{
			UserId: 123,
			Id:     456,
		}

		mockPostService.On("DeletePost", mock.Anything, int64(123), int64(456)).
			Return(custom_errors.ErrForbidden)

		resp, err := handler.DeletePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, statusErr.Code())
		assert.Equal(t, "user is not the author", statusErr.Message())
		mockPostService.AssertExpectations(t)
	})

	t.Run("MediaQueryError", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewDeletePostHandler(mockPostService, validate, testLogger)

		req := &pb.DeletePostRequest{
			UserId: 123,
			Id:     456,
		}

		mockPostService.On("DeletePost", mock.Anything, int64(123), int64(456)).
			Return(custom_errors.ErrMediaQueryFailed)

		resp, err := handler.DeletePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, statusErr.Code())
		assert.Equal(t, "failed to query media", statusErr.Message())
		mockPostService.AssertExpectations(t)
	})

	t.Run("MediaDetachError", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewDeletePostHandler(mockPostService, validate, testLogger)

		req := &pb.DeletePostRequest{
			UserId: 123,
			Id:     456,
		}

		mockPostService.On("DeletePost", mock.Anything, int64(123), int64(456)).
			Return(custom_errors.ErrMediaDetachFailed)

		resp, err := handler.DeletePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, statusErr.Code())
		assert.Equal(t, "failed to detach media", statusErr.Message())
		mockPostService.AssertExpectations(t)
	})

	t.Run("TagQueryError", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewDeletePostHandler(mockPostService, validate, testLogger)

		req := &pb.DeletePostRequest{
			UserId: 123,
			Id:     456,
		}

		mockPostService.On("DeletePost", mock.Anything, int64(123), int64(456)).
			Return(custom_errors.ErrTagQueryFailed)

		resp, err := handler.DeletePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, statusErr.Code())
		assert.Equal(t, "failed to query tags", statusErr.Message())
		mockPostService.AssertExpectations(t)
	})

	t.Run("TagDeleteError", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewDeletePostHandler(mockPostService, validate, testLogger)

		req := &pb.DeletePostRequest{
			UserId: 123,
			Id:     456,
		}

		mockPostService.On("DeletePost", mock.Anything, int64(123), int64(456)).
			Return(custom_errors.ErrTagDeleteFailed)

		resp, err := handler.DeletePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, statusErr.Code())
		assert.Equal(t, "failed to remove tags", statusErr.Message())
		mockPostService.AssertExpectations(t)
	})

	t.Run("DatabaseQueryError", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewDeletePostHandler(mockPostService, validate, testLogger)

		req := &pb.DeletePostRequest{
			UserId: 123,
			Id:     456,
		}

		mockPostService.On("DeletePost", mock.Anything, int64(123), int64(456)).
			Return(custom_errors.ErrDatabaseQuery)

		resp, err := handler.DeletePost(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, statusErr.Code())
		assert.Equal(t, "database error", statusErr.Message())
		mockPostService.AssertExpectations(t)
	})

	t.Run("InternalError", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewDeletePostHandler(mockPostService, validate, testLogger)

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
		assert.Equal(t, "failed to delete post", statusErr.Message())
		mockPostService.AssertExpectations(t)
	})
}
