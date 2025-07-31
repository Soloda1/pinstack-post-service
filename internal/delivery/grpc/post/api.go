package post_grpc

import (
	"context"
	"pinstack-post-service/internal/logger"
	post_service "pinstack-post-service/internal/service/post"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/go-playground/validator/v10"
	pb "github.com/soloda1/pinstack-proto-definitions/gen/go/pinstack-proto-definitions/post/v1"
)

const MaxMediaPosition = 9
const MinMediaPosition = 1

var validate = validator.New()

type PostGRPCService struct {
	pb.UnimplementedPostServiceServer
	postService       post_service.Service
	log               *logger.Logger
	createPostHandler *CreatePostHandler
	getPostHandler    *GetPostHandler
	listPostsHandler  *ListPostsHandler
	updatePostHandler *UpdatePostHandler
	deletePostHandler *DeletePostHandler
}

func NewPostGRPCService(postService *post_service.PostService, log *logger.Logger) *PostGRPCService {
	createPostHandler := NewCreatePostHandler(postService, validate, log)
	getPostHandler := NewGetPostHandler(postService, validate, log)
	listPostsHandler := NewListPostsHandler(postService, validate, log)
	updatePostHandler := NewUpdatePostHandler(postService, validate, log)
	deletePostHandler := NewDeletePostHandler(postService, validate, log)
	return &PostGRPCService{
		postService:       postService,
		log:               log,
		createPostHandler: createPostHandler,
		getPostHandler:    getPostHandler,
		listPostsHandler:  listPostsHandler,
		updatePostHandler: updatePostHandler,
		deletePostHandler: deletePostHandler,
	}
}

func (s *PostGRPCService) CreatePost(ctx context.Context, req *pb.CreatePostRequest) (*pb.Post, error) {
	return s.createPostHandler.CreatePost(ctx, req)
}

func (s *PostGRPCService) GetPost(ctx context.Context, req *pb.GetPostRequest) (*pb.Post, error) {
	return s.getPostHandler.GetPost(ctx, req)
}

func (s *PostGRPCService) ListPosts(ctx context.Context, req *pb.ListPostsRequest) (*pb.ListPostsResponse, error) {
	return s.listPostsHandler.ListPosts(ctx, req)
}

func (s *PostGRPCService) UpdatePost(ctx context.Context, req *pb.UpdatePostRequest) (*pb.Post, error) {
	return s.updatePostHandler.UpdatePost(ctx, req)
}

func (s *PostGRPCService) DeletePost(ctx context.Context, req *pb.DeletePostRequest) (*emptypb.Empty, error) {
	return s.deletePostHandler.DeletePost(ctx, req)
}
