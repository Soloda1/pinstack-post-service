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
	"google.golang.org/protobuf/types/known/timestamppb"

	model "pinstack-post-service/internal/domain/models"
)

type PostGetter interface {
	GetPostByID(ctx context.Context, id int64) (*model.PostDetailed, error)
}

type GetPostHandler struct {
	pb.UnimplementedPostServiceServer
	postService PostGetter
	validate    *validator.Validate
	log         ports.Logger
}

func NewGetPostHandler(postService PostGetter, validate *validator.Validate, log ports.Logger) *GetPostHandler {
	return &GetPostHandler{
		postService: postService,
		validate:    validate,
		log:         log,
	}
}

type GetPostRequestInternal struct {
	PostID int64 `validate:"required,gt=0"`
}

func (h *GetPostHandler) GetPost(ctx context.Context, req *pb.GetPostRequest) (*pb.Post, error) {
	h.log.Debug("Handling GetPost request", slog.Int64("post_id", req.GetId()))

	validationReq := &GetPostRequestInternal{
		PostID: req.GetId(),
	}

	if err := h.validate.Struct(validationReq); err != nil {
		h.log.Debug("GetPost validation failed", slog.Int64("post_id", req.GetId()), slog.String("error", err.Error()))
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	h.log.Debug("Getting post by ID", slog.Int64("post_id", req.GetId()))
	retrievedPostModel, err := h.postService.GetPostByID(ctx, req.GetId())
	if err != nil {
		switch {
		case errors.Is(err, custom_errors.ErrPostNotFound):
			h.log.Debug("Post not found", slog.Int64("post_id", req.GetId()))
			return nil, status.Error(codes.NotFound, "post not found")
		case errors.Is(err, custom_errors.ErrPostValidation):
			h.log.Debug("Post retrieval validation failed", slog.Int64("post_id", req.GetId()), slog.String("error", err.Error()))
			return nil, status.Error(codes.InvalidArgument, "post retrieval validation failed")
		default:
			h.log.Error("Failed to get post", slog.Int64("post_id", req.GetId()), slog.String("error", err.Error()))
			return nil, status.Error(codes.Internal, "failed to get post")
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

	h.log.Debug("Post retrieved successfully",
		slog.Int64("post_id", postID),
		slog.Int64("author_id", authorID),
		slog.Int("tags_count", len(pbTags)),
		slog.Int("media_count", len(pbMedia)))

	return resp, nil
}
