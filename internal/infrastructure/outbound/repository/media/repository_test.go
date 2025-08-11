package media_repository_test

import (
	"context"
	media_repository "pinstack-post-service/internal/domain/ports/output/media"
	"testing"

	"github.com/soloda1/pinstack-proto-definitions/custom_errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	model "pinstack-post-service/internal/domain/models"
	"pinstack-post-service/internal/infrastructure/logger"
	"pinstack-post-service/internal/infrastructure/outbound/repository/media/memory"
)

func setupMediaTest(t *testing.T) (media_repository.Repository, func()) {
	log := logger.New("test")
	repo := memory.NewMediaRepository(log)
	return repo, func() {}
}

func TestMediaRepository_Attach(t *testing.T) {
	repo, cleanup := setupMediaTest(t)
	defer cleanup()

	mediaRepo, ok := repo.(*memory.MediaRepository)
	require.True(t, ok)

	postID := int64(1)
	mediaRepo.SimulatePostExists(postID, true)

	tests := []struct {
		name    string
		postID  int64
		media   []*model.PostMedia
		wantErr error
	}{
		{
			name:   "successful attach",
			postID: postID,
			media: []*model.PostMedia{
				{
					URL:      "https://example.com/image1.jpg",
					Type:     model.MediaTypeImage,
					Position: 1,
				},
				{
					URL:      "https://example.com/image2.jpg",
					Type:     model.MediaTypeImage,
					Position: 2,
				},
			},
			wantErr: nil,
		},
		{
			name:   "post not found",
			postID: 999, // Non-existent post
			media: []*model.PostMedia{
				{
					URL:      "https://example.com/image.jpg",
					Type:     model.MediaTypeImage,
					Position: 1,
				},
			},
			wantErr: custom_errors.ErrPostNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Attach(context.Background(), tt.postID, tt.media)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)

				media, err := repo.GetByPost(context.Background(), tt.postID)
				assert.NoError(t, err)
				assert.Len(t, media, len(tt.media))

				for i, m := range tt.media {
					assert.Equal(t, m.URL, media[i].URL)
					assert.Equal(t, m.Type, media[i].Type)
					assert.Equal(t, m.Position, media[i].Position)
					assert.NotZero(t, media[i].ID)
					assert.True(t, media[i].CreatedAt.Valid)
					assert.NotZero(t, media[i].CreatedAt.Time)
				}
			}
		})
	}
}

func TestMediaRepository_Reorder(t *testing.T) {
	repo, cleanup := setupMediaTest(t)
	defer cleanup()

	// Convert to MediaRepository to use the SimulatePostExists method
	mediaRepo, ok := repo.(*memory.MediaRepository)
	require.True(t, ok)

	// Simulate post exists
	postID := int64(1)
	mediaRepo.SimulatePostExists(postID, true)

	media := []*model.PostMedia{
		{
			URL:      "https://example.com/image1.jpg",
			Type:     model.MediaTypeImage,
			Position: 1,
		},
		{
			URL:      "https://example.com/image2.jpg",
			Type:     model.MediaTypeImage,
			Position: 2,
		},
		{
			URL:      "https://example.com/image3.jpg",
			Type:     model.MediaTypeImage,
			Position: 3,
		},
	}

	err := repo.Attach(context.Background(), postID, media)
	require.NoError(t, err)

	attachedMedia, err := repo.GetByPost(context.Background(), postID)
	require.NoError(t, err)
	require.Len(t, attachedMedia, 3)

	tests := []struct {
		name         string
		postID       int64
		newPositions map[int64]int
		wantErr      error
	}{
		{
			name:   "successful reorder",
			postID: postID,
			newPositions: map[int64]int{
				attachedMedia[0].ID: 3, // Move first to last
				attachedMedia[1].ID: 1, // Move second to first
				attachedMedia[2].ID: 2, // Move third to middle
			},
			wantErr: nil,
		},
		{
			name:   "post not found",
			postID: 999, // Non-existent post
			newPositions: map[int64]int{
				attachedMedia[0].ID: 1,
			},
			wantErr: custom_errors.ErrMediaReorderFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Reorder(context.Background(), tt.postID, tt.newPositions)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)

				reorderedMedia, err := repo.GetByPost(context.Background(), tt.postID)
				assert.NoError(t, err)

				for _, media := range reorderedMedia {
					if newPos, ok := tt.newPositions[media.ID]; ok {
						assert.Equal(t, int32(newPos), media.Position)
					}
				}

				for i := 0; i < len(reorderedMedia)-1; i++ {
					assert.LessOrEqual(t, reorderedMedia[i].Position, reorderedMedia[i+1].Position)
				}
			}
		})
	}
}

func TestMediaRepository_Detach(t *testing.T) {
	repo, cleanup := setupMediaTest(t)
	defer cleanup()

	// Convert to MediaRepository to use the SimulatePostExists method
	mediaRepo, ok := repo.(*memory.MediaRepository)
	require.True(t, ok)

	// Simulate post exists
	postID := int64(1)
	mediaRepo.SimulatePostExists(postID, true)

	// First attach some media
	media := []*model.PostMedia{
		{
			URL:      "https://example.com/image1.jpg",
			Type:     model.MediaTypeImage,
			Position: 1,
		},
		{
			URL:      "https://example.com/image2.jpg",
			Type:     model.MediaTypeImage,
			Position: 2,
		},
	}

	err := repo.Attach(context.Background(), postID, media)
	require.NoError(t, err)

	// Get the attached media to get their IDs
	attachedMedia, err := repo.GetByPost(context.Background(), postID)
	require.NoError(t, err)
	require.Len(t, attachedMedia, 2)

	tests := []struct {
		name     string
		mediaIDs []int64
		wantErr  error
	}{
		{
			name:     "detach single media",
			mediaIDs: []int64{attachedMedia[0].ID},
			wantErr:  nil,
		},
		{
			name:     "detach multiple media",
			mediaIDs: []int64{attachedMedia[0].ID, attachedMedia[1].ID},
			wantErr:  nil,
		},
		{
			name:     "detach non-existent media",
			mediaIDs: []int64{999}, // Non-existent media
			wantErr:  nil,          // The repository doesn't return error for non-existent media
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Detach(context.Background(), tt.mediaIDs)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)

				remainingMedia, err := repo.GetByPost(context.Background(), postID)
				assert.NoError(t, err)

				for _, mediaID := range tt.mediaIDs {
					for _, m := range remainingMedia {
						assert.NotEqual(t, mediaID, m.ID, "Media should have been detached")
					}
				}
			}
		})
	}
}

func TestMediaRepository_GetByPost(t *testing.T) {
	repo, cleanup := setupMediaTest(t)
	defer cleanup()

	// Convert to MediaRepository to use the SimulatePostExists method
	mediaRepo, ok := repo.(*memory.MediaRepository)
	require.True(t, ok)

	postID1 := int64(1)
	postID2 := int64(2)
	mediaRepo.SimulatePostExists(postID1, true)
	mediaRepo.SimulatePostExists(postID2, true)

	post1Media := []*model.PostMedia{
		{
			URL:      "https://example.com/post1-image1.jpg",
			Type:     model.MediaTypeImage,
			Position: 1,
		},
		{
			URL:      "https://example.com/post1-image2.jpg",
			Type:     model.MediaTypeImage,
			Position: 2,
		},
	}
	err := repo.Attach(context.Background(), postID1, post1Media)
	require.NoError(t, err)

	post2Media := []*model.PostMedia{
		{
			URL:      "https://example.com/post2-image1.jpg",
			Type:     model.MediaTypeImage,
			Position: 1,
		},
	}
	err = repo.Attach(context.Background(), postID2, post2Media)
	require.NoError(t, err)

	tests := []struct {
		name    string
		postID  int64
		wantLen int
		wantErr error
	}{
		{
			name:    "post with multiple media",
			postID:  postID1,
			wantLen: 2,
			wantErr: nil,
		},
		{
			name:    "post with single media",
			postID:  postID2,
			wantLen: 1,
			wantErr: nil,
		},
		{
			name:    "post with no media",
			postID:  999, // Non-existent post
			wantLen: 0,
			wantErr: nil, // GetByPost returns empty slice for non-existent post
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetByPost(context.Background(), tt.postID)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Len(t, got, tt.wantLen)

				if tt.postID == postID1 && tt.wantLen > 0 {

					assert.Contains(t, []string{"https://example.com/post1-image1.jpg", "https://example.com/post1-image2.jpg"}, got[0].URL)
				} else if tt.postID == postID2 && tt.wantLen > 0 {

					assert.Equal(t, "https://example.com/post2-image1.jpg", got[0].URL)
				}
			}
		})
	}
}

func TestMediaRepository_GetByPosts(t *testing.T) {
	repo, cleanup := setupMediaTest(t)
	defer cleanup()

	// Convert to MediaRepository to use the SimulatePostExists method
	mediaRepo, ok := repo.(*memory.MediaRepository)
	require.True(t, ok)

	// Simulate posts exist
	postID1 := int64(1)
	postID2 := int64(2)
	postID3 := int64(3)
	mediaRepo.SimulatePostExists(postID1, true)
	mediaRepo.SimulatePostExists(postID2, true)
	mediaRepo.SimulatePostExists(postID3, true)

	// Attach media to post 1
	post1Media := []*model.PostMedia{
		{
			URL:      "https://example.com/post1-image1.jpg",
			Type:     model.MediaTypeImage,
			Position: 1,
		},
		{
			URL:      "https://example.com/post1-image2.jpg",
			Type:     model.MediaTypeImage,
			Position: 2,
		},
	}
	err := repo.Attach(context.Background(), postID1, post1Media)
	require.NoError(t, err)

	// Attach media to post 2
	post2Media := []*model.PostMedia{
		{
			URL:      "https://example.com/post2-image1.jpg",
			Type:     model.MediaTypeImage,
			Position: 1,
		},
	}
	err = repo.Attach(context.Background(), postID2, post2Media)
	require.NoError(t, err)

	tests := []struct {
		name     string
		postIDs  []int64
		wantKeys []int64
		wantErr  error
	}{
		{
			name:     "multiple posts with media",
			postIDs:  []int64{postID1, postID2},
			wantKeys: []int64{postID1, postID2},
			wantErr:  nil,
		},
		{
			name:     "one post with media, one without",
			postIDs:  []int64{postID1, postID3},
			wantKeys: []int64{postID1}, // Only post1 should have media
			wantErr:  nil,
		},
		{
			name:     "posts with no media",
			postIDs:  []int64{postID3, 999}, // Post 3 and non-existent post
			wantKeys: []int64{},             // No posts should have media
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetByPosts(context.Background(), tt.postIDs)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)

				for _, wantKey := range tt.wantKeys {
					assert.Contains(t, got, wantKey)
				}
				assert.Len(t, got, len(tt.wantKeys))

				if len(tt.wantKeys) > 0 {
					if media, ok := got[postID1]; ok {
						assert.Len(t, media, 2)
					}

					if media, ok := got[postID2]; ok {
						assert.Len(t, media, 1)
						assert.Equal(t, "https://example.com/post2-image1.jpg", media[0].URL)
					}
				}
			}
		})
	}
}
