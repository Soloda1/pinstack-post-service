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

type PostCreator interface {
	CreatePost(ctx context.Context, post *model.CreatePostDTO) (*model.PostDetailed, error)
}

type CreatePostHandler struct {
	pb.UnimplementedPostServiceServer
	postService PostCreator
	validate    *validator.Validate
}

func NewCreatePostHandler(postService PostCreator, validate *validator.Validate) *CreatePostHandler {
	return &CreatePostHandler{
		postService: postService,
		validate:    validate,
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
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	dtoMediaItems := make([]*model.PostMediaInput, 0, len(req.GetMedia()))
	for i, m := range req.GetMedia() {
		position := m.GetPosition()
		if position < 1 || position > 9 {
			position = int32(i + 1)
			if position > 9 {
				continue
			}
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
		switch err {
		case custom_errors.ErrPostValidation:
			return nil, status.Errorf(codes.InvalidArgument, "post creation validation failed: %v", err)
		default:
			return nil, status.Errorf(codes.Internal, "failed to create post: %v", err)
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

	return resp, nil
}
