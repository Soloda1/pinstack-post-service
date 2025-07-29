package post_grpc

import (
	"context"
	"errors"

	"github.com/go-playground/validator/v10"
	pb "github.com/soloda1/pinstack-proto-definitions/gen/go/pinstack-proto-definitions/post/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"pinstack-post-service/internal/custom_errors"
	"pinstack-post-service/internal/model"
)

type PostUpdater interface {
	UpdatePost(ctx context.Context, userID int64, id int64, post *model.UpdatePostDTO) error
	GetPostByID(ctx context.Context, id int64) (*model.PostDetailed, error)
}

type UpdatePostHandler struct {
	pb.UnimplementedPostServiceServer
	postService PostUpdater
	validate    *validator.Validate
}

func NewUpdatePostHandler(postService PostUpdater, validate *validator.Validate) *UpdatePostHandler {
	return &UpdatePostHandler{
		postService: postService,
		validate:    validate,
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
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	dtoMediaItems := make([]*model.PostMediaInput, 0, len(req.GetMedia()))
	for i, m := range req.GetMedia() {

		position := m.GetPosition()
		if position < MinMediaPosition || position > MaxMediaPosition {
			position = int32(i + 1)
			if position > MinMediaPosition {
				continue
			}
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
		switch {
		case errors.Is(err, custom_errors.ErrPostNotFound):
			return nil, status.Errorf(codes.NotFound, "post not found: %v", err)
		case errors.Is(err, custom_errors.ErrPostValidation):
			return nil, status.Errorf(codes.InvalidArgument, "post update validation failed: %v", err)
		case errors.Is(err, custom_errors.ErrInvalidInput):
			return nil, status.Errorf(codes.PermissionDenied, "user is not author of post: %v", err)
		default:
			return nil, status.Errorf(codes.Internal, "failed to update post: %v", err)
		}
	}

	updatedPost, err := h.postService.GetPostByID(ctx, req.GetId())
	if err != nil {
		switch {
		case errors.Is(err, custom_errors.ErrPostNotFound):
			return nil, status.Errorf(codes.NotFound, "post not found: %v", err)
		case errors.Is(err, custom_errors.ErrPostValidation):
			return nil, status.Errorf(codes.InvalidArgument, "post update validation failed: %v", err)
		default:
			return nil, status.Errorf(codes.Internal, "failed to update post: %v", err)
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

	return resp, nil
}
