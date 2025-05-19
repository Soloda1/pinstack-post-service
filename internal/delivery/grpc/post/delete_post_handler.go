package post_grpc

import (
	"context"
	"errors"

	"github.com/go-playground/validator/v10"
	pb "github.com/soloda1/pinstack-proto-definitions/gen/go/pinstack-proto-definitions/post/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"pinstack-post-service/internal/custom_errors"
)

type PostDeleter interface {
	DeletePost(ctx context.Context, id int64) error
}

type DeletePostHandler struct {
	pb.UnimplementedPostServiceServer
	postService PostDeleter
	validate    *validator.Validate
}

func NewDeletePostHandler(postService PostDeleter, validate *validator.Validate) *DeletePostHandler {
	return &DeletePostHandler{
		postService: postService,
		validate:    validate,
	}
}

type DeletePostRequestInternal struct {
	Id int64 `validate:"required,gt=0"`
}

func (h *DeletePostHandler) DeletePost(ctx context.Context, req *pb.DeletePostRequest) (*emptypb.Empty, error) {
	validationReq := &DeletePostRequestInternal{
		Id: req.GetId(),
	}

	if err := h.validate.Struct(validationReq); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	err := h.postService.DeletePost(ctx, req.GetId())
	if err != nil {
		switch {
		case errors.Is(err, custom_errors.ErrPostNotFound):
			return nil, status.Errorf(codes.NotFound, "post not found: %v", err)
		case errors.Is(err, custom_errors.ErrPostValidation):
			return nil, status.Errorf(codes.InvalidArgument, "post delete validation failed: %v", err)
		default:
			return nil, status.Errorf(codes.Internal, "failed to delete post: %v", err)
		}
	}

	return &emptypb.Empty{}, nil
}
