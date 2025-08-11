package tag_repository_test

import (
	"context"
	tag_repository "pinstack-post-service/internal/domain/ports/output/tag"
	"testing"

	"github.com/soloda1/pinstack-proto-definitions/custom_errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"pinstack-post-service/internal/infrastructure/logger"
	"pinstack-post-service/internal/infrastructure/outbound/repository/tag/memory"
)

func setupTagTest(t *testing.T) (tag_repository.Repository, func()) {
	log := logger.New("test")
	repo := memory.NewTagRepository(log)
	return repo, func() {}
}

func TestTagRepository_Create(t *testing.T) {
	repo, cleanup := setupTagTest(t)
	defer cleanup()

	tests := []struct {
		name    string
		tagName string
		wantErr error
	}{
		{
			name:    "successful creation",
			tagName: "testtag",
			wantErr: nil,
		},
		{
			name:    "duplicate tag name",
			tagName: "testtag",
			wantErr: nil, // In-memory implementation allows re-creating same tag
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.Create(context.Background(), tt.tagName)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.tagName, got.Name)
				assert.NotZero(t, got.ID)
			}
		})
	}
}

func TestTagRepository_FindByNames(t *testing.T) {
	repo, cleanup := setupTagTest(t)
	defer cleanup()

	tagNames := []string{"tag1", "tag2", "tag3"}
	for _, name := range tagNames {
		_, err := repo.Create(context.Background(), name)
		require.NoError(t, err)
	}

	tests := []struct {
		name     string
		tagNames []string
		wantLen  int
		wantErr  error
	}{
		{
			name:     "find multiple existing tags",
			tagNames: []string{"tag1", "tag2"},
			wantLen:  2,
			wantErr:  nil,
		},
		{
			name:     "find one existing tag",
			tagNames: []string{"tag3"},
			wantLen:  1,
			wantErr:  nil,
		},
		{
			name:     "find non-existent tag",
			tagNames: []string{"nonexistent"},
			wantLen:  0,
			wantErr:  nil,
		},
		{
			name:     "find mix of existing and non-existing tags",
			tagNames: []string{"tag1", "nonexistent"},
			wantLen:  1,
			wantErr:  nil,
		},
		{
			name:     "empty tag names",
			tagNames: []string{},
			wantLen:  0,
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.FindByNames(context.Background(), tt.tagNames)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Len(t, got, tt.wantLen)

				for _, tag := range got {
					assert.Contains(t, tt.tagNames, tag.Name)
				}
			}
		})
	}
}

func TestTagRepository_FindByPost(t *testing.T) {
	repo, cleanup := setupTagTest(t)
	defer cleanup()

	tagRepo, ok := repo.(*memory.TagRepository)
	require.True(t, ok)

	postID := int64(1)
	tagRepo.SimulatePostExists(postID, true)

	tagNames := []string{"tag1", "tag2", "tag3"}
	err := repo.TagPost(context.Background(), postID, tagNames)
	require.NoError(t, err)

	tests := []struct {
		name    string
		postID  int64
		wantLen int
		wantErr error
	}{
		{
			name:    "post with tags",
			postID:  postID,
			wantLen: 3,
			wantErr: nil,
		},
		{
			name:    "post without tags",
			postID:  999, // Non-existent post
			wantLen: 0,
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.FindByPost(context.Background(), tt.postID)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Len(t, got, tt.wantLen)

				if tt.postID == postID {

					for _, tag := range got {
						assert.Contains(t, tagNames, tag.Name)
					}
				}
			}
		})
	}
}

func TestTagRepository_DeleteUnused(t *testing.T) {
	repo, cleanup := setupTagTest(t)
	defer cleanup()

	// Convert to TagRepository to use the SimulatePostExists method
	tagRepo, ok := repo.(*memory.TagRepository)
	require.True(t, ok)

	postID := int64(1)
	tagRepo.SimulatePostExists(postID, true)

	_, err := repo.Create(context.Background(), "tag1")
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), "tag2")
	require.NoError(t, err)

	err = repo.TagPost(context.Background(), postID, []string{"tag1"})
	require.NoError(t, err)

	t.Run("delete unused tags", func(t *testing.T) {
		err := repo.DeleteUnused(context.Background())
		assert.NoError(t, err)

		tags, err := repo.FindByNames(context.Background(), []string{"tag1", "tag2"})
		assert.NoError(t, err)
		assert.Len(t, tags, 1)
		assert.Equal(t, "tag1", tags[0].Name)
	})
}

func TestTagRepository_TagPost(t *testing.T) {
	repo, cleanup := setupTagTest(t)
	defer cleanup()

	// Convert to TagRepository to use the SimulatePostExists method
	tagRepo, ok := repo.(*memory.TagRepository)
	require.True(t, ok)

	postID := int64(1)
	tagRepo.SimulatePostExists(postID, true)

	tests := []struct {
		name     string
		postID   int64
		tagNames []string
		wantErr  error
	}{
		{
			name:     "tag existing post with new tags",
			postID:   postID,
			tagNames: []string{"tag1", "tag2"},
			wantErr:  nil,
		},
		{
			name:     "tag non-existent post",
			postID:   999,
			tagNames: []string{"tag3"},
			wantErr:  custom_errors.ErrPostNotFound,
		},
		{
			name:     "tag with empty tags",
			postID:   postID,
			tagNames: []string{},
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.TagPost(context.Background(), tt.postID, tt.tagNames)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)

				if len(tt.tagNames) > 0 {
					tags, err := repo.FindByPost(context.Background(), tt.postID)
					assert.NoError(t, err)

					tagMap := make(map[string]bool)
					for _, tag := range tags {
						tagMap[tag.Name] = true
					}

					for _, name := range tt.tagNames {
						assert.True(t, tagMap[name], "Tag %s should be associated with post", name)
					}
				}
			}
		})
	}
}

func TestTagRepository_UntagPost(t *testing.T) {
	repo, cleanup := setupTagTest(t)
	defer cleanup()

	// Convert to TagRepository to use the SimulatePostExists method
	tagRepo, ok := repo.(*memory.TagRepository)
	require.True(t, ok)

	postID := int64(1)
	tagRepo.SimulatePostExists(postID, true)

	initialTags := []string{"tag1", "tag2", "tag3"}
	err := repo.TagPost(context.Background(), postID, initialTags)
	require.NoError(t, err)

	tests := []struct {
		name       string
		postID     int64
		tagNames   []string
		wantRemain []string
		wantErr    error
	}{
		{
			name:       "untag single tag",
			postID:     postID,
			tagNames:   []string{"tag1"},
			wantRemain: []string{"tag2", "tag3"},
			wantErr:    nil,
		},
		{
			name:       "untag multiple tags",
			postID:     postID,
			tagNames:   []string{"tag1", "tag2", "tag3"},
			wantRemain: []string{},
			wantErr:    nil,
		},
		{
			name:       "untag non-existent tag",
			postID:     postID,
			tagNames:   []string{"nonexistent"},
			wantRemain: []string{"tag1", "tag2", "tag3"},
			wantErr:    nil,
		},
		{
			name:     "untag from non-existent post",
			postID:   999,
			tagNames: []string{"tag1"},
			wantErr:  custom_errors.ErrPostNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.name != "untag single tag" {
				err := repo.TagPost(context.Background(), postID, initialTags)
				require.NoError(t, err)
			}

			err := repo.UntagPost(context.Background(), tt.postID, tt.tagNames)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)

				tags, err := repo.FindByPost(context.Background(), tt.postID)
				assert.NoError(t, err)

				assert.Len(t, tags, len(tt.wantRemain))

				tagMap := make(map[string]bool)
				for _, tag := range tags {
					tagMap[tag.Name] = true
				}

				for _, name := range tt.wantRemain {
					assert.True(t, tagMap[name], "Tag %s should still be associated with post", name)
				}
			}
		})
	}
}

func TestTagRepository_ReplacePostTags(t *testing.T) {
	repo, cleanup := setupTagTest(t)
	defer cleanup()

	// Convert to TagRepository to use the SimulatePostExists method
	tagRepo, ok := repo.(*memory.TagRepository)
	require.True(t, ok)

	postID := int64(1)
	tagRepo.SimulatePostExists(postID, true)

	// First tag the post
	initialTags := []string{"tag1", "tag2", "tag3"}
	err := repo.TagPost(context.Background(), postID, initialTags)
	require.NoError(t, err)

	tests := []struct {
		name     string
		postID   int64
		newTags  []string
		wantTags []string
		wantErr  error
	}{
		{
			name:     "replace with new tags",
			postID:   postID,
			newTags:  []string{"newtag1", "newtag2"},
			wantTags: []string{"newtag1", "newtag2"},
			wantErr:  nil,
		},
		{
			name:     "replace with empty tags",
			postID:   postID,
			newTags:  []string{},
			wantTags: []string{},
			wantErr:  nil,
		},
		{
			name:     "replace on non-existent post",
			postID:   999,
			newTags:  []string{"tag1"},
			wantTags: []string{},
			wantErr:  custom_errors.ErrPostNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.name != "replace with new tags" {
				err := repo.TagPost(context.Background(), postID, initialTags)
				require.NoError(t, err)
			}

			err := repo.ReplacePostTags(context.Background(), tt.postID, tt.newTags)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)

				tags, err := repo.FindByPost(context.Background(), tt.postID)
				assert.NoError(t, err)

				assert.Len(t, tags, len(tt.wantTags))

				tagMap := make(map[string]bool)
				for _, tag := range tags {
					tagMap[tag.Name] = true
				}

				for _, name := range tt.wantTags {
					assert.True(t, tagMap[name], "Tag %s should be associated with post", name)
				}

				for _, name := range initialTags {
					if !contains(tt.wantTags, name) {
						assert.False(t, tagMap[name], "Tag %s should not be associated with post", name)
					}
				}
			}
		})
	}
}

func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}
