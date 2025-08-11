package post_grpc

import (
	"context"
	"errors"
	"log/slog"
	ports "pinstack-post-service/internal/domain/ports/output"

	"github.com/soloda1/pinstack-proto-definitions/custom_errors"

	"github.com/go-playground/validator/v10"
	pb "github.com/soloda1/pinstack-proto-definitions/gen/go/pinstack-proto-definitions/post/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type PostDeleter interface {
	DeletePost(ctx context.Context, userID int64, id int64) error
}

type DeletePostHandler struct {
	pb.UnimplementedPostServiceServer
	postService PostDeleter
	validate    *validator.Validate
	log         ports.Logger
}

func NewDeletePostHandler(postService PostDeleter, validate *validator.Validate, log ports.Logger) *DeletePostHandler {
	return &DeletePostHandler{
		postService: postService,
		validate:    validate,
		log:         log,
	}
}

type DeletePostRequestInternal struct {
	Id int64 `validate:"required,gt=0"`
}

func (h *DeletePostHandler) DeletePost(ctx context.Context, req *pb.DeletePostRequest) (*emptypb.Empty, error) {
	h.log.Debug("Received DeletePost request",
		slog.Int64("post_id", req.GetId()),
		slog.Int64("user_id", req.GetUserId()))

	validationReq := &DeletePostRequestInternal{
		Id: req.GetId(),
	}

	if err := h.validate.Struct(validationReq); err != nil {
		h.log.Debug("Request validation failed",
			slog.Int64("post_id", req.GetId()),
			slog.Int64("user_id", req.GetUserId()),
			slog.String("error", err.Error()))
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	err := h.postService.DeletePost(ctx, req.GetUserId(), req.GetId())
	if err != nil {
		h.log.Debug("Error deleting post",
			slog.Int64("post_id", req.GetId()),
			slog.Int64("user_id", req.GetUserId()),
			slog.String("error", err.Error()))

		switch {
		case errors.Is(err, custom_errors.ErrPostNotFound):
			return nil, status.Error(codes.NotFound, "post not found")
		case errors.Is(err, custom_errors.ErrPostValidation):
			return nil, status.Error(codes.InvalidArgument, "validation failed")
		case errors.Is(err, custom_errors.ErrForbidden):
			return nil, status.Error(codes.PermissionDenied, "user is not the author")
		case errors.Is(err, custom_errors.ErrMediaQueryFailed):
			h.log.Error("Failed to query media", slog.Int64("post_id", req.GetId()), slog.String("error", err.Error()))
			return nil, status.Error(codes.Internal, "failed to query media")
		case errors.Is(err, custom_errors.ErrMediaDetachFailed):
			h.log.Error("Failed to detach media", slog.Int64("post_id", req.GetId()), slog.String("error", err.Error()))
			return nil, status.Error(codes.Internal, "failed to detach media")
		case errors.Is(err, custom_errors.ErrTagQueryFailed):
			h.log.Error("Failed to query tags", slog.Int64("post_id", req.GetId()), slog.String("error", err.Error()))
			return nil, status.Error(codes.Internal, "failed to query tags")
		case errors.Is(err, custom_errors.ErrTagDeleteFailed):
			h.log.Error("Failed to remove tags", slog.Int64("post_id", req.GetId()), slog.String("error", err.Error()))
			return nil, status.Error(codes.Internal, "failed to remove tags")
		case errors.Is(err, custom_errors.ErrDatabaseQuery):
			h.log.Error("Database error", slog.Int64("post_id", req.GetId()), slog.String("error", err.Error()))
			return nil, status.Error(codes.Internal, "database error")
		default:
			h.log.Error("Unexpected error deleting post", slog.Int64("post_id", req.GetId()), slog.String("error", err.Error()))
			return nil, status.Error(codes.Internal, "failed to delete post")
		}
	}

	h.log.Debug("Post deleted successfully",
		slog.Int64("post_id", req.GetId()),
		slog.Int64("user_id", req.GetUserId()))
	return &emptypb.Empty{}, nil
}
