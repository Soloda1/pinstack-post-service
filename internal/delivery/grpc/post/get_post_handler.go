package post_grpc

import (
	"context"

	"github.com/go-playground/validator/v10"
	pb "github.com/soloda1/pinstack-proto-definitions/gen/go/pinstack-proto-definitions/post/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"pinstack-post-service/internal/custom_errors"
	"pinstack-post-service/internal/model"
)

type PostGetter interface {
	GetPostByID(ctx context.Context, id int64) (*model.PostDetailed, error)
}

type GetPostHandler struct {
	pb.UnimplementedPostServiceServer
	postService PostGetter
	validate    *validator.Validate
}

func NewGetPostHandler(postService PostGetter, validate *validator.Validate) *GetPostHandler {
	return &GetPostHandler{
		postService: postService,
		validate:    validate,
	}
}

type GetPostRequestInternal struct {
	PostID int64 `validate:"required,gt=0"`
}

func (h *GetPostHandler) GetPost(ctx context.Context, req *pb.GetPostRequest) (*pb.Post, error) {
	validationReq := &GetPostRequestInternal{
		PostID: req.GetId(),
	}

	if err := h.validate.Struct(validationReq); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	retrievedPostModel, err := h.postService.GetPostByID(ctx, req.GetId())
	if err != nil {
		switch err {
		case custom_errors.ErrPostNotFound:
			return nil, status.Errorf(codes.NotFound, "post not found: %v", err)
		case custom_errors.ErrPostValidation: // Assuming validation might also occur on retrieval path
			return nil, status.Errorf(codes.InvalidArgument, "post retrieval validation failed: %v", err)
		default:
			return nil, status.Errorf(codes.Internal, "failed to get post: %v", err)
		}
	}

	pbMedia := make([]*pb.Media, len(retrievedPostModel.Media))
	for i, m := range retrievedPostModel.Media {
		var mediaCreatedAtPb *timestamppb.Timestamp
		if m.CreatedAt.Valid {
			mediaCreatedAtPb = timestamppb.New(m.CreatedAt.Time)
		}
		pbMedia[i] = &pb.Media{
			Id:        m.ID,
			Url:       m.URL,
			Type:      string(m.Type),
			Position:  m.Position,
			CreatedAt: mediaCreatedAtPb,
		}
	}

	var postID int64
	var authorID int64
	var title string
	var content string
	var createdAtPb *timestamppb.Timestamp
	var updatedAtPb *timestamppb.Timestamp

	if retrievedPostModel.Post != nil {
		postID = retrievedPostModel.Post.ID
		authorID = retrievedPostModel.Post.AuthorID
		title = retrievedPostModel.Post.Title
		if retrievedPostModel.Post.Content != nil {
			content = *retrievedPostModel.Post.Content
		}
		if retrievedPostModel.Post.CreatedAt.Valid {
			createdAtPb = timestamppb.New(retrievedPostModel.Post.CreatedAt.Time)
		}
		if retrievedPostModel.Post.UpdatedAt.Valid {
			updatedAtPb = timestamppb.New(retrievedPostModel.Post.UpdatedAt.Time)
		}
	}

	pbTags := make([]string, len(retrievedPostModel.Tags))
	for i, t := range retrievedPostModel.Tags {
		pbTags[i] = t.Name
	}

	resp := &pb.Post{
		Id:        postID,
		AuthorId:  authorID,
		Title:     title,
		Content:   content,
		Tags:      pbTags,
		Media:     pbMedia,
		CreatedAt: createdAtPb,
		UpdatedAt: updatedAtPb,
	}

	return resp, nil
}
