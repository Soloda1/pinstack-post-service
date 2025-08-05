package post_repository_test

import (
	"context"
	"github.com/soloda1/pinstack-proto-definitions/custom_errors"
	post_repository "pinstack-post-service/internal/repository/post"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"pinstack-post-service/internal/logger"
	"pinstack-post-service/internal/model"
	"pinstack-post-service/internal/repository/post/memory"
)

func setupPostTest(t *testing.T) (post_repository.Repository, func()) {
	log := logger.New("test")
	repo := memory.NewPostRepository(log)
	return repo, func() {}
}

func TestPostRepository_Create(t *testing.T) {
	repo, cleanup := setupPostTest(t)
	defer cleanup()

	content := "Test content"
	tests := []struct {
		name    string
		post    *model.Post
		wantErr error
	}{
		{
			name: "successful creation",
			post: &model.Post{
				AuthorID: 1,
				Title:    "Test Post",
				Content:  &content,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.Create(context.Background(), tt.post)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.post.Title, got.Title)
				assert.Equal(t, tt.post.AuthorID, got.AuthorID)
				assert.Equal(t, tt.post.Content, got.Content)
				assert.NotZero(t, got.ID)
				assert.True(t, got.CreatedAt.Valid)
				assert.True(t, got.UpdatedAt.Valid)
			}
		})
	}
}

func TestPostRepository_GetByID(t *testing.T) {
	repo, cleanup := setupPostTest(t)
	defer cleanup()

	content := "Test content"
	post := &model.Post{
		AuthorID: 1,
		Title:    "Test Post",
		Content:  &content,
	}
	created, err := repo.Create(context.Background(), post)
	require.NoError(t, err)
	require.NotNil(t, created)

	tests := []struct {
		name    string
		id      int64
		want    *model.Post
		wantErr error
	}{
		{
			name:    "successful get",
			id:      created.ID,
			want:    created,
			wantErr: nil,
		},
		{
			name:    "post not found",
			id:      999,
			want:    nil,
			wantErr: custom_errors.ErrPostNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetByID(context.Background(), tt.id)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.want.ID, got.ID)
				assert.Equal(t, tt.want.AuthorID, got.AuthorID)
				assert.Equal(t, tt.want.Title, got.Title)
				assert.Equal(t, tt.want.Content, got.Content)
			}
		})
	}
}

func TestPostRepository_GetByAuthor(t *testing.T) {
	repo, cleanup := setupPostTest(t)
	defer cleanup()

	content1 := "Content 1"
	content2 := "Content 2"

	posts := []*model.Post{
		{
			AuthorID: 1,
			Title:    "Author 1 Post 1",
			Content:  &content1,
		},
		{
			AuthorID: 1,
			Title:    "Author 1 Post 2",
			Content:  &content2,
		},
		{
			AuthorID: 2,
			Title:    "Author 2 Post",
			Content:  &content1,
		},
	}

	for _, p := range posts {
		_, err := repo.Create(context.Background(), p)
		require.NoError(t, err)
	}

	tests := []struct {
		name     string
		authorID int64
		wantLen  int
		wantErr  error
	}{
		{
			name:     "author with multiple posts",
			authorID: 1,
			wantLen:  2,
			wantErr:  nil,
		},
		{
			name:     "author with one post",
			authorID: 2,
			wantLen:  1,
			wantErr:  nil,
		},
		{
			name:     "author with no posts",
			authorID: 3,
			wantLen:  0,
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetByAuthor(context.Background(), tt.authorID)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Len(t, got, tt.wantLen)
				for _, post := range got {
					assert.Equal(t, tt.authorID, post.AuthorID)
				}
			}
		})
	}
}

func TestPostRepository_Update(t *testing.T) {
	repo, cleanup := setupPostTest(t)
	defer cleanup()

	content := "Original content"
	post := &model.Post{
		AuthorID: 1,
		Title:    "Original Title",
		Content:  &content,
	}
	created, err := repo.Create(context.Background(), post)
	require.NoError(t, err)
	require.NotNil(t, created)

	newTitle := "Updated Title"
	newContent := "Updated content"

	tests := []struct {
		name    string
		id      int64
		update  *model.UpdatePostDTO
		wantErr error
	}{
		{
			name: "update title only",
			id:   created.ID,
			update: &model.UpdatePostDTO{
				Title: &newTitle,
			},
			wantErr: nil,
		},
		{
			name: "update content only",
			id:   created.ID,
			update: &model.UpdatePostDTO{
				Content: &newContent,
			},
			wantErr: nil,
		},
		{
			name: "update both title and content",
			id:   created.ID,
			update: &model.UpdatePostDTO{
				Title:   &newTitle,
				Content: &newContent,
			},
			wantErr: nil,
		},
		{
			name: "post not found",
			id:   999,
			update: &model.UpdatePostDTO{
				Title: &newTitle,
			},
			wantErr: custom_errors.ErrPostNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.Update(context.Background(), tt.id, tt.update)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.id, got.ID)

				if tt.update.Title != nil {
					assert.Equal(t, *tt.update.Title, got.Title)
				}

				if tt.update.Content != nil {
					assert.Equal(t, *tt.update.Content, *got.Content)
				}

				assert.True(t, got.UpdatedAt.Valid)
			}
		})
	}
}

func TestPostRepository_Delete(t *testing.T) {
	repo, cleanup := setupPostTest(t)
	defer cleanup()

	content := "Test content"
	post := &model.Post{
		AuthorID: 1,
		Title:    "Test Post",
		Content:  &content,
	}
	created, err := repo.Create(context.Background(), post)
	require.NoError(t, err)
	require.NotNil(t, created)

	tests := []struct {
		name    string
		id      int64
		wantErr error
	}{
		{
			name:    "successful delete",
			id:      created.ID,
			wantErr: nil,
		},
		{
			name:    "post not found",
			id:      999,
			wantErr: custom_errors.ErrPostNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Delete(context.Background(), tt.id)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
				_, getErr := repo.GetByID(context.Background(), tt.id)
				assert.Error(t, getErr)
				assert.Equal(t, custom_errors.ErrPostNotFound, getErr)
			}
		})
	}
}

func TestPostRepository_List(t *testing.T) {
	repo, cleanup := setupPostTest(t)
	defer cleanup()

	content := "Test content"

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	yesterdayTS := pgtype.Timestamp{Time: yesterday, Valid: true}
	tomorrowTS := pgtype.Timestamp{Time: tomorrow, Valid: true}

	posts := []*model.Post{
		{
			AuthorID: 1,
			Title:    "Post 1",
			Content:  &content,
		},
		{
			AuthorID: 1,
			Title:    "Post 2",
			Content:  &content,
		},
		{
			AuthorID: 2,
			Title:    "Post 3",
			Content:  &content,
		},
	}

	for _, p := range posts {
		_, err := repo.Create(context.Background(), p)
		require.NoError(t, err)
	}

	authorID := int64(1)
	limit := 2
	offset := 0

	tests := []struct {
		name    string
		filters model.PostFilters
		wantLen int
		wantErr error
	}{
		{
			name: "filter by author",
			filters: model.PostFilters{
				AuthorID: &authorID,
			},
			wantLen: 2,
			wantErr: nil,
		},
		{
			name: "filter by created after",
			filters: model.PostFilters{
				CreatedAfter: (*pgtype.Timestamptz)(&yesterdayTS),
			},
			wantLen: 3,
			wantErr: nil,
		},
		{
			name: "filter by created before",
			filters: model.PostFilters{
				CreatedBefore: (*pgtype.Timestamptz)(&tomorrowTS),
			},
			wantLen: 3,
			wantErr: nil,
		},
		{
			name: "pagination",
			filters: model.PostFilters{
				Limit:  &limit,
				Offset: &offset,
			},
			wantLen: 2,
			wantErr: nil,
		},
		{
			name:    "no filters",
			filters: model.PostFilters{},
			wantLen: 3,
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, total, err := repo.List(context.Background(), tt.filters)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
				assert.Nil(t, got)
				assert.Zero(t, total)
			} else {
				assert.NoError(t, err)
				assert.Len(t, got, tt.wantLen)
				assert.True(t, total >= len(got))

				if len(got) > 1 {
					for i := 0; i < len(got)-1; i++ {
						assert.True(t, got[i].CreatedAt.Time.After(got[i+1].CreatedAt.Time) ||
							got[i].CreatedAt.Time.Equal(got[i+1].CreatedAt.Time))
					}
				}

				if tt.filters.AuthorID != nil {
					for _, p := range got {
						assert.Equal(t, *tt.filters.AuthorID, p.AuthorID)
					}
				}
			}
		})
	}
}
