package post_grpc

import (
	"context"

	"pinstack-post-service/internal/model"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
	pb "github.com/soloda1/pinstack-proto-definitions/gen/go/pinstack-proto-definitions/post/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PostLister interface {
	ListPosts(ctx context.Context, filters *model.PostFilters) ([]*model.PostDetailed, int, error)
}

type ListPostsHandler struct {
	pb.UnimplementedPostServiceServer
	postService PostLister
	validate    *validator.Validate
}

func NewListPostsHandler(postService PostLister, validate *validator.Validate) *ListPostsHandler {
	return &ListPostsHandler{
		postService: postService,
		validate:    validate,
	}
}

type ListPostsRequestInternal struct {
	AuthorID *int64 `validate:"omitempty,gt=0"`
	Offset   *int   `validate:"omitempty,gte=0"`
	Limit    *int   `validate:"omitempty,gt=0,lte=100"`
}

func (h *ListPostsHandler) ListPosts(ctx context.Context, req *pb.ListPostsRequest) (*pb.ListPostsResponse, error) {
	var authorIDPtr *int64
	if req.AuthorId != 0 {
		authorIDPtr = &req.AuthorId
	}
	var offsetPtr *int
	if req.Offset != 0 {
		offset := int(req.Offset)
		offsetPtr = &offset
	}
	var limitPtr *int
	if req.Limit != 0 {
		limit := int(req.Limit)
		limitPtr = &limit
	}

	validationReq := &ListPostsRequestInternal{
		AuthorID: authorIDPtr,
		Offset:   offsetPtr,
		Limit:    limitPtr,
	}

	if err := h.validate.Struct(validationReq); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	filters := &model.PostFilters{
		AuthorID: authorIDPtr,
		Limit:    limitPtr,
		Offset:   offsetPtr,
	}

	if req.CreatedAfter != nil {
		createdAfter := pgtype.Timestamptz{
			Time:  req.CreatedAfter.AsTime(),
			Valid: true,
		}
		filters.CreatedAfter = &createdAfter
	}

	if req.CreatedBefore != nil {
		createdBefore := pgtype.Timestamptz{
			Time:  req.CreatedBefore.AsTime(),
			Valid: true,
		}
		filters.CreatedBefore = &createdBefore
	}

	if len(req.TagNames) > 0 {
		filters.TagNames = req.TagNames
	}

	posts, total, err := h.postService.ListPosts(ctx, filters)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list posts: %v", err)
	}

	pbPosts := make([]*pb.Post, len(posts))
	for i, post := range posts {
		var pbMedia []*pb.Media
		if post.Media != nil {
			pbMedia = make([]*pb.Media, len(post.Media))
			for j, m := range post.Media {
				var mediaCreatedAtPb *timestamppb.Timestamp
				if m.CreatedAt.Valid {
					mediaCreatedAtPb = timestamppb.New(m.CreatedAt.Time)
				}
				pbMedia[j] = &pb.Media{
					Id:        m.ID,
					Url:       m.URL,
					Type:      string(m.Type),
					Position:  m.Position,
					CreatedAt: mediaCreatedAtPb,
				}
			}
		}

		var postID int64
		var authorID int64
		var title string
		var content string
		var createdAtPb *timestamppb.Timestamp
		var updatedAtPb *timestamppb.Timestamp

		if post.Post != nil {
			postID = post.Post.ID
			authorID = post.Post.AuthorID
			title = post.Post.Title
			if post.Post.Content != nil {
				content = *post.Post.Content
			}
			if post.Post.CreatedAt.Valid {
				createdAtPb = timestamppb.New(post.Post.CreatedAt.Time)
			}
			if post.Post.UpdatedAt.Valid {
				updatedAtPb = timestamppb.New(post.Post.UpdatedAt.Time)
			}
		}

		pbTags := make([]string, len(post.Tags))
		for k, t := range post.Tags {
			pbTags[k] = t.Name
		}

		pbPosts[i] = &pb.Post{
			Id:        postID,
			AuthorId:  authorID,
			Title:     title,
			Content:   content,
			Tags:      pbTags,
			Media:     pbMedia,
			CreatedAt: createdAtPb,
			UpdatedAt: updatedAtPb,
		}
	}

	resp := &pb.ListPostsResponse{
		Posts: pbPosts,
		Total: int64(total),
	}

	return resp, nil
}
