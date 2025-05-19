package post_grpc

import (
	"context"
	"pinstack-post-service/internal/logger"
	post_service "pinstack-post-service/internal/service/post"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/go-playground/validator/v10"
	pb "github.com/soloda1/pinstack-proto-definitions/gen/go/pinstack-proto-definitions/post/v1"
)

var validate = validator.New()

type PostGRPCService struct {
	pb.UnimplementedPostServiceServer
	postService       post_service.Service
	log               *logger.Logger
	createPostHandler *CreatePostHandler
	getPostHandler    *GetPostHandler
}

func NewPostGRPCService(postService *post_service.PostService, log *logger.Logger) *PostGRPCService {
	createPostHandler := NewCreatePostHandler(postService, validate)
	getPostHandler := NewGetPostHandler(postService, validate)
	return &PostGRPCService{
		postService:       postService,
		log:               log,
		createPostHandler: createPostHandler,
		getPostHandler:    getPostHandler,
	}
}

func (s *PostGRPCService) CreatePost(ctx context.Context, req *pb.CreatePostRequest) (*pb.Post, error) {
	return s.createPostHandler.CreatePost(ctx, req)
}

func (s *PostGRPCService) GetPost(ctx context.Context, req *pb.GetPostRequest) (*pb.Post, error) {
	return s.getPostHandler.GetPost(ctx, req)
}

func (s *PostGRPCService) ListPosts(ctx context.Context, req *pb.ListPostsRequest) (*pb.ListPostsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListPosts not implemented")
}

func (s *PostGRPCService) UpdatePost(ctx context.Context, req *pb.UpdatePostRequest) (*pb.Post, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdatePost not implemented")
}

func (s *PostGRPCService) DeletePost(ctx context.Context, req *pb.DeletePostRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeletePost not implemented")
}
