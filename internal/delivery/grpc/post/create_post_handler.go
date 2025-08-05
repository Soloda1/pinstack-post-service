package post_grpc

import (
	"context"
	"github.com/soloda1/pinstack-proto-definitions/custom_errors"
	"log/slog"

	"github.com/go-playground/validator/v10"
	pb "github.com/soloda1/pinstack-proto-definitions/gen/go/pinstack-proto-definitions/post/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"pinstack-post-service/internal/logger"
	"pinstack-post-service/internal/model"
)

type PostCreator interface {
	CreatePost(ctx context.Context, post *model.CreatePostDTO) (*model.PostDetailed, error)
}

type CreatePostHandler struct {
	pb.UnimplementedPostServiceServer
	postService PostCreator
	validate    *validator.Validate
	log         *logger.Logger
}

func NewCreatePostHandler(postService PostCreator, validate *validator.Validate, log *logger.Logger) *CreatePostHandler {
	return &CreatePostHandler{
		postService: postService,
		validate:    validate,
		log:         log,
	}
}

type CreatePostRequestInternal struct {
	AuthorID int64                 `validate:"required"`
	Title    string                `validate:"required,min=3,max=255"`
	Content  string                `validate:"required,min=10"`
	Tags     []string              `validate:"omitempty,dive,min=2,max=50"`
	Media    []*MediaInputInternal `validate:"omitempty,max=9,dive"`
}

type MediaInputInternal struct {
	URL      string `validate:"required,url"`
	Type     string `validate:"required,oneof=image video"`
	Position int32  `validate:"gte=1,lte=9"`
}

func (h *CreatePostHandler) CreatePost(ctx context.Context, req *pb.CreatePostRequest) (*pb.Post, error) {
	h.log.Debug("Received CreatePost request",
		slog.Int64("author_id", req.GetAuthorId()),
		slog.String("title", req.GetTitle()),
		slog.Bool("has_content", req.Content != ""),
		slog.Int("media_items_count", len(req.GetMedia())),
		slog.Int("tags_count", len(req.GetTags())))

	internalMedia := make([]*MediaInputInternal, len(req.GetMedia()))
	for i, m := range req.GetMedia() {
		internalMedia[i] = &MediaInputInternal{
			URL:      m.GetUrl(),
			Type:     m.GetType(),
			Position: m.GetPosition(),
		}
	}

	validationReq := &CreatePostRequestInternal{
		AuthorID: req.GetAuthorId(),
		Title:    req.GetTitle(),
		Content:  req.GetContent(),
		Tags:     req.GetTags(),
		Media:    internalMedia,
	}

	if err := h.validate.Struct(validationReq); err != nil {
		h.log.Debug("Request validation failed",
			slog.Int64("author_id", req.GetAuthorId()),
			slog.String("error", err.Error()))
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	dtoMediaItems := make([]*model.PostMediaInput, 0, len(req.GetMedia()))
	for i, m := range req.GetMedia() {
		position := m.GetPosition()
		if position < MinMediaPosition || position > MaxMediaPosition {
			h.log.Debug("Invalid media position, adjusting",
				slog.Int("original_position", int(position)),
				slog.Int("index", i),
				slog.String("url", m.GetUrl()))

			position = int32(i + 1)

			if position > MaxMediaPosition {
				h.log.Debug("Skipping media item due to position constraints",
					slog.Int("adjusted_position", int(position)),
					slog.Int("max_allowed", MaxMediaPosition),
					slog.String("url", m.GetUrl()))
				continue
			}

			h.log.Debug("Media position adjusted",
				slog.Int("new_position", int(position)),
				slog.String("url", m.GetUrl()))
		}
		dtoMediaItems = append(dtoMediaItems, &model.PostMediaInput{
			URL:      m.GetUrl(),
			Type:     model.MediaType(m.GetType()),
			Position: position,
		})
	}

	postDTO := &model.CreatePostDTO{
		AuthorID:   req.GetAuthorId(),
		Title:      req.GetTitle(),
		Content:    &req.Content,
		Tags:       req.GetTags(),
		MediaItems: dtoMediaItems,
	}

	createdPostModel, err := h.postService.CreatePost(ctx, postDTO)
	if err != nil {
		h.log.Debug("Error creating post",
			slog.Int64("author_id", req.GetAuthorId()),
			slog.String("error", err.Error()))

		switch err {
		case custom_errors.ErrPostValidation:
			return nil, status.Error(codes.InvalidArgument, "validation failed")
		default:
			h.log.Error("Unexpected error creating post",
				slog.Int64("author_id", req.GetAuthorId()),
				slog.String("error", err.Error()))
			return nil, status.Error(codes.Internal, "internal service error")
		}
	}

	pbMedia := make([]*pb.Media, len(createdPostModel.Media))
	for i, m := range createdPostModel.Media {
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

	if createdPostModel.Post != nil {
		postID = createdPostModel.Post.ID
		authorID = createdPostModel.Post.AuthorID
		title = createdPostModel.Post.Title
		content = *createdPostModel.Post.Content
		if createdPostModel.Post.CreatedAt.Valid {
			createdAtPb = timestamppb.New(createdPostModel.Post.CreatedAt.Time)
		}
		if createdPostModel.Post.UpdatedAt.Valid {
			updatedAtPb = timestamppb.New(createdPostModel.Post.UpdatedAt.Time)
		}
	}

	pbTags := make([]string, len(createdPostModel.Tags))
	for i, t := range createdPostModel.Tags {
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

	h.log.Debug("Post created successfully",
		slog.Int64("post_id", postID),
		slog.Int64("author_id", authorID),
		slog.Int("tags_count", len(pbTags)),
		slog.Int("media_count", len(pbMedia)))

	return resp, nil
}
