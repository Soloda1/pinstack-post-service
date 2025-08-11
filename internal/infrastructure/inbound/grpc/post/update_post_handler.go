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

type PostUpdater interface {
	UpdatePost(ctx context.Context, userID int64, id int64, post *model.UpdatePostDTO) error
	GetPostByID(ctx context.Context, id int64) (*model.PostDetailed, error)
}

type UpdatePostHandler struct {
	pb.UnimplementedPostServiceServer
	postService PostUpdater
	validate    *validator.Validate
	log         ports.Logger
}

func NewUpdatePostHandler(postService PostUpdater, validate *validator.Validate, log ports.Logger) *UpdatePostHandler {
	return &UpdatePostHandler{
		postService: postService,
		validate:    validate,
		log:         log,
	}
}

type UpdatePostRequestInternal struct {
	Id      int64                 `validate:"required,gt=0"`
	Title   *string               `validate:"omitempty"`
	Content *string               `validate:"omitempty"`
	Tags    []string              `validate:"omitempty,dive"`
	Media   []*MediaInputInternal `validate:"omitempty,dive"`
}

func (h *UpdatePostHandler) UpdatePost(ctx context.Context, req *pb.UpdatePostRequest) (*pb.Post, error) {
	h.log.Debug("Received UpdatePost request",
		slog.Int64("post_id", req.GetId()),
		slog.Int64("user_id", req.GetUserId()),
		slog.Bool("has_title_update", req.Title != ""),
		slog.Bool("has_content_update", req.Content != ""),
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

	validationReq := &UpdatePostRequestInternal{
		Id:      req.GetId(),
		Title:   &req.Title,
		Content: &req.Content,
		Tags:    req.GetTags(),
		Media:   internalMedia,
	}

	if err := h.validate.Struct(validationReq); err != nil {
		h.log.Debug("Request validation failed",
			slog.Int64("post_id", req.GetId()),
			slog.Int64("user_id", req.GetUserId()),
			slog.String("error", err.Error()))
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
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

	updateDTO := &model.UpdatePostDTO{
		UserID:     req.GetUserId(),
		Title:      &req.Title,
		Content:    &req.Content,
		Tags:       req.GetTags(),
		MediaItems: dtoMediaItems,
	}

	err := h.postService.UpdatePost(ctx, req.GetUserId(), req.GetId(), updateDTO)
	if err != nil {
		h.log.Debug("Error updating post", slog.Int64("id", req.GetId()), slog.Int64("user_id", req.GetUserId()), slog.String("error", err.Error()))
		switch {
		case errors.Is(err, custom_errors.ErrPostNotFound):
			return nil, status.Error(codes.NotFound, custom_errors.ErrPostNotFound.Error())
		case errors.Is(err, custom_errors.ErrPostValidation):
			return nil, status.Error(codes.InvalidArgument, custom_errors.ErrPostValidation.Error())
		case errors.Is(err, custom_errors.ErrForbidden) || errors.Is(err, custom_errors.ErrInvalidInput):
			// Map both ErrForbidden and ErrInvalidInput to PermissionDenied for consistency with API gateway
			return nil, status.Error(codes.PermissionDenied, custom_errors.ErrForbidden.Error())
		default:
			h.log.Error("Unexpected error updating post", slog.Int64("id", req.GetId()), slog.String("error", err.Error()))
			return nil, status.Error(codes.Internal, custom_errors.ErrInternalServiceError.Error())
		}
	}

	updatedPost, err := h.postService.GetPostByID(ctx, req.GetId())
	if err != nil {
		h.log.Debug("Error getting updated post", slog.Int64("id", req.GetId()), slog.String("error", err.Error()))
		switch {
		case errors.Is(err, custom_errors.ErrPostNotFound):
			return nil, status.Error(codes.NotFound, custom_errors.ErrPostNotFound.Error())
		case errors.Is(err, custom_errors.ErrPostValidation):
			return nil, status.Error(codes.InvalidArgument, custom_errors.ErrPostValidation.Error())
		case errors.Is(err, custom_errors.ErrForbidden):
			return nil, status.Error(codes.PermissionDenied, custom_errors.ErrForbidden.Error())
		default:
			h.log.Error("Unexpected error retrieving updated post", slog.Int64("id", req.GetId()), slog.String("error", err.Error()))
			return nil, status.Error(codes.Internal, custom_errors.ErrInternalServiceError.Error())
		}
	}

	pbMedia := make([]*pb.Media, len(updatedPost.Media))
	for i, m := range updatedPost.Media {
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

	if updatedPost.Post != nil {
		postID = updatedPost.Post.ID
		authorID = updatedPost.Post.AuthorID
		title = updatedPost.Post.Title
		if updatedPost.Post.Content != nil {
			content = *updatedPost.Post.Content
		}
		if updatedPost.Post.CreatedAt.Valid {
			createdAtPb = timestamppb.New(updatedPost.Post.CreatedAt.Time)
		}
		if updatedPost.Post.UpdatedAt.Valid {
			updatedAtPb = timestamppb.New(updatedPost.Post.UpdatedAt.Time)
		}
	}

	pbTags := make([]string, len(updatedPost.Tags))
	for i, t := range updatedPost.Tags {
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

	h.log.Debug("Successfully updated post",
		slog.Int64("post_id", postID),
		slog.Int64("author_id", authorID),
		slog.Int("tags_count", len(pbTags)),
		slog.Int("media_count", len(pbMedia)))
	return resp, nil
}
