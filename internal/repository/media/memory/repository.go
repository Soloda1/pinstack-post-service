package memory

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"pinstack-post-service/internal/custom_errors"
	"pinstack-post-service/internal/logger"
	"pinstack-post-service/internal/model"
)

type MediaRepository struct {
	log           *logger.Logger
	mu            sync.RWMutex
	mediaByPostID map[int64][]*model.PostMedia
	mediaByID     map[int64]*model.PostMedia
	postExists    map[int64]bool
	nextID        int64
}

func NewMediaRepository(log *logger.Logger) *MediaRepository {
	return &MediaRepository{
		log:           log,
		mediaByPostID: make(map[int64][]*model.PostMedia),
		mediaByID:     make(map[int64]*model.PostMedia),
		postExists:    make(map[int64]bool),
		nextID:        1,
	}
}

func (m *MediaRepository) SimulatePostExists(postID int64, exists bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.postExists[postID] = exists
}

func (m *MediaRepository) Attach(ctx context.Context, postID int64, media []*model.PostMedia) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if exists, found := m.postExists[postID]; !found || !exists {
		m.log.Warn("Post not found during media attach", slog.Int64("post_id", postID))
		return custom_errors.ErrPostNotFound
	}

	for _, md := range media {
		newMedia := &model.PostMedia{
			ID:       m.nextID,
			URL:      md.URL,
			Type:     md.Type,
			Position: md.Position,
			CreatedAt: pgtype.Timestamptz{
				Time:  time.Now(),
				Valid: true,
			},
		}
		m.nextID++

		m.mediaByID[newMedia.ID] = newMedia
		m.mediaByPostID[postID] = append(m.mediaByPostID[postID], newMedia)
	}

	sort.Slice(m.mediaByPostID[postID], func(i, j int) bool {
		return m.mediaByPostID[postID][i].Position < m.mediaByPostID[postID][j].Position
	})

	return nil
}

func (m *MediaRepository) Reorder(ctx context.Context, postID int64, newPositions map[int64]int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.mediaByPostID[postID]; !exists {
		return custom_errors.ErrMediaReorderFailed
	}

	for mediaID, newPosition := range newPositions {
		if media, exists := m.mediaByID[mediaID]; exists {
			media.Position = int32(newPosition)
		}
	}

	sort.Slice(m.mediaByPostID[postID], func(i, j int) bool {
		return m.mediaByPostID[postID][i].Position < m.mediaByPostID[postID][j].Position
	})

	return nil
}

func (m *MediaRepository) Detach(ctx context.Context, mediaIDs []int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, mediaID := range mediaIDs {
		if media, exists := m.mediaByID[mediaID]; exists {
			for postID, postMedia := range m.mediaByPostID {
				var updatedMedia []*model.PostMedia
				for _, pm := range postMedia {
					if pm.ID != mediaID {
						updatedMedia = append(updatedMedia, pm)
					}
				}
				m.mediaByPostID[postID] = updatedMedia
			}
			delete(m.mediaByID, media.ID)
		}
	}

	return nil
}

func (m *MediaRepository) GetByPost(ctx context.Context, postID int64) ([]*model.PostMedia, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if media, exists := m.mediaByPostID[postID]; exists {
		result := make([]*model.PostMedia, len(media))
		for i, item := range media {
			copy := *item
			result[i] = &copy
		}
		return result, nil
	}

	return []*model.PostMedia{}, nil
}

func (m *MediaRepository) GetByPosts(ctx context.Context, postIDs []int64) (map[int64][]*model.PostMedia, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[int64][]*model.PostMedia)

	for _, postID := range postIDs {
		if media, exists := m.mediaByPostID[postID]; exists {
			mediaCopy := make([]*model.PostMedia, len(media))
			for i, item := range media {
				copy := *item
				mediaCopy[i] = &copy
			}
			result[postID] = mediaCopy
		}
	}

	return result, nil
}
