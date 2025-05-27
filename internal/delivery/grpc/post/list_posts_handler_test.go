package post_grpc_test

import (
	"context"
	"errors"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
	pb "github.com/soloda1/pinstack-proto-definitions/gen/go/pinstack-proto-definitions/post/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	post_grpc "pinstack-post-service/internal/delivery/grpc/post"
	"pinstack-post-service/internal/model"
	mockpost "pinstack-post-service/mocks/post"
	"testing"
	"time"
)

func TestListPostsHandler_ListPosts(t *testing.T) {
	validate := validator.New()

	t.Run("Success_WithAllFilters", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewListPostsHandler(mockPostService, validate)

		req := &pb.ListPostsRequest{
			AuthorId: 123,
			Limit:    10,
			Offset:   20,
		}

		createdAt := time.Now()
		updatedAt := time.Now()
		content := "Test post content"

		expectedPosts := []*model.PostDetailed{
			{
				Post: &model.Post{
					ID:        1,
					AuthorID:  123,
					Title:     "First Post",
					Content:   &content,
					CreatedAt: pgtype.Timestamp{Time: createdAt, Valid: true},
					UpdatedAt: pgtype.Timestamp{Time: updatedAt, Valid: true},
				},
				Media: []*model.PostMedia{
					{
						ID:        1,
						PostID:    1,
						URL:       "https://example.com/image1.jpg",
						Type:      "image",
						Position:  0,
						CreatedAt: pgtype.Timestamptz{Time: createdAt, Valid: true},
					},
				},
				Tags: []*model.Tag{
					{ID: 1, Name: "tag1"},
					{ID: 2, Name: "tag2"},
				},
			},
			{
				Post: &model.Post{
					ID:        2,
					AuthorID:  123,
					Title:     "Second Post",
					Content:   &content,
					CreatedAt: pgtype.Timestamp{Time: createdAt, Valid: true},
					UpdatedAt: pgtype.Timestamp{Time: updatedAt, Valid: true},
				},
				Media: []*model.PostMedia{
					{
						ID:        2,
						PostID:    2,
						URL:       "https://example.com/image2.jpg",
						Type:      "image",
						Position:  0,
						CreatedAt: pgtype.Timestamptz{Time: createdAt, Valid: true},
					},
				},
				Tags: []*model.Tag{
					{ID: 2, Name: "tag2"},
					{ID: 3, Name: "tag3"},
				},
			},
		}

		mockPostService.On("ListPosts", mock.Anything, mock.MatchedBy(func(filters *model.PostFilters) bool {
			return filters != nil &&
				filters.AuthorID != nil && *filters.AuthorID == req.AuthorId &&
				filters.Limit != nil && *filters.Limit == int(req.Limit) &&
				filters.Offset != nil && *filters.Offset == int(req.Offset)
		})).Return(expectedPosts, nil)

		resp, err := handler.ListPosts(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int64(len(expectedPosts)), resp.Total)
		assert.Len(t, resp.Posts, len(expectedPosts))

		for i, post := range resp.Posts {
			assert.Equal(t, expectedPosts[i].Post.ID, post.Id)
			assert.Equal(t, expectedPosts[i].Post.AuthorID, post.AuthorId)
			assert.Equal(t, expectedPosts[i].Post.Title, post.Title)
			assert.Equal(t, *expectedPosts[i].Post.Content, post.Content)

			assert.NotNil(t, post.CreatedAt)
			assert.Equal(t, timestamppb.New(createdAt).Seconds, post.CreatedAt.Seconds)
			assert.NotNil(t, post.UpdatedAt)
			assert.Equal(t, timestamppb.New(updatedAt).Seconds, post.UpdatedAt.Seconds)

			assert.Equal(t, len(expectedPosts[i].Tags), len(post.Tags))
			for j, tag := range expectedPosts[i].Tags {
				assert.Equal(t, tag.Name, post.Tags[j])
			}

			assert.Equal(t, len(expectedPosts[i].Media), len(post.Media))
			for j, media := range expectedPosts[i].Media {
				assert.Equal(t, media.ID, post.Media[j].Id)
				assert.Equal(t, media.URL, post.Media[j].Url)
				assert.Equal(t, string(media.Type), post.Media[j].Type)
				assert.Equal(t, media.Position, post.Media[j].Position)
			}
		}

		mockPostService.AssertExpectations(t)
	})

	t.Run("Success_EmptyFilters", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewListPostsHandler(mockPostService, validate)

		req := &pb.ListPostsRequest{}

		expectedPosts := []*model.PostDetailed{}

		mockPostService.On("ListPosts", mock.Anything, mock.MatchedBy(func(filters *model.PostFilters) bool {
			return filters != nil &&
				filters.AuthorID == nil &&
				filters.Limit == nil &&
				filters.Offset == nil
		})).Return(expectedPosts, nil)

		resp, err := handler.ListPosts(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int64(0), resp.Total)
		assert.Empty(t, resp.Posts)
		mockPostService.AssertExpectations(t)
	})

	t.Run("Success_OnlyAuthorIDFilter", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewListPostsHandler(mockPostService, validate)

		req := &pb.ListPostsRequest{
			AuthorId: 123,
		}

		expectedPosts := []*model.PostDetailed{}

		mockPostService.On("ListPosts", mock.Anything, mock.MatchedBy(func(filters *model.PostFilters) bool {
			return filters != nil &&
				filters.AuthorID != nil && *filters.AuthorID == req.AuthorId &&
				filters.Limit == nil &&
				filters.Offset == nil
		})).Return(expectedPosts, nil)

		resp, err := handler.ListPosts(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int64(0), resp.Total)
		assert.Empty(t, resp.Posts)
		mockPostService.AssertExpectations(t)
	})

	t.Run("ValidationError_NegativeOffset", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewListPostsHandler(mockPostService, validate)

		req := &pb.ListPostsRequest{
			AuthorId: 123,
			Offset:   -10,
			Limit:    20,
		}

		resp, err := handler.ListPosts(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())
		assert.Contains(t, statusErr.Message(), "invalid request")

		mockPostService.AssertNotCalled(t, "ListPosts")
	})

	t.Run("ValidationError_TooLargeLimit", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewListPostsHandler(mockPostService, validate)

		req := &pb.ListPostsRequest{
			Limit: 500,
		}

		resp, err := handler.ListPosts(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())
		assert.Contains(t, statusErr.Message(), "invalid request")

		mockPostService.AssertNotCalled(t, "ListPosts")
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewListPostsHandler(mockPostService, validate)

		req := &pb.ListPostsRequest{
			Limit:  10,
			Offset: 0,
		}

		mockPostService.On("ListPosts", mock.Anything, mock.Anything).
			Return(nil, errors.New("database error"))

		resp, err := handler.ListPosts(context.Background(), req)

		assert.Nil(t, resp)
		assert.Error(t, err)

		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, statusErr.Code())
		assert.Contains(t, statusErr.Message(), "failed to list posts")

		mockPostService.AssertExpectations(t)
	})

	t.Run("Success_WithNullableFields", func(t *testing.T) {
		mockPostService := new(mockpost.Service)
		handler := post_grpc.NewListPostsHandler(mockPostService, validate)

		req := &pb.ListPostsRequest{
			Limit: 10,
		}

		expectedPosts := []*model.PostDetailed{
			{
				Post: &model.Post{
					ID:        1,
					AuthorID:  123,
					Title:     "Post with nullable fields",
					Content:   nil,
					CreatedAt: pgtype.Timestamp{Valid: false},
					UpdatedAt: pgtype.Timestamp{Valid: false},
				},
				Media: nil,
				Tags:  nil,
			},
		}

		mockPostService.On("ListPosts", mock.Anything, mock.Anything).Return(expectedPosts, nil)

		resp, err := handler.ListPosts(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int64(1), resp.Total)
		assert.Len(t, resp.Posts, 1)

		post := resp.Posts[0]
		assert.Equal(t, expectedPosts[0].Post.ID, post.Id)
		assert.Equal(t, expectedPosts[0].Post.AuthorID, post.AuthorId)
		assert.Equal(t, expectedPosts[0].Post.Title, post.Title)
		assert.Empty(t, post.Content)
		assert.Nil(t, post.CreatedAt)
		assert.Nil(t, post.UpdatedAt)
		assert.Empty(t, post.Media)
		assert.Empty(t, post.Tags)

		mockPostService.AssertExpectations(t)
	})
}
