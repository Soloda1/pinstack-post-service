package media_repository_postgres

import (
	"context"
	"log/slog"
	model "pinstack-post-service/internal/domain/models"
	ports "pinstack-post-service/internal/domain/ports/output"
	"pinstack-post-service/internal/infrastructure/outbound/repository/postgres/db"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/soloda1/pinstack-proto-definitions/custom_errors"
)

type MediaRepository struct {
	log     ports.Logger
	db      db.PgDB
	metrics ports.MetricsProvider
}

func NewMediaRepository(db db.PgDB, log ports.Logger, metrics ports.MetricsProvider) *MediaRepository {
	return &MediaRepository{db: db, log: log, metrics: metrics}
}

func (m *MediaRepository) Attach(ctx context.Context, postID int64, media []*model.PostMedia) error {
	start := time.Now()
	var exists bool
	err := m.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM posts WHERE id = @post_id)`, pgx.NamedArgs{"post_id": postID}).Scan(&exists)
	if err != nil {
		m.metrics.IncrementDatabaseQueries("media_attach", false)
		m.metrics.RecordDatabaseQueryDuration("media_attach", time.Since(start))
		m.log.Error("Failed to get post by id in Attach media", slog.Int64("post_id", postID), slog.String("err", err.Error()))
		return custom_errors.ErrDatabaseQuery
	}
	if !exists {
		m.metrics.IncrementDatabaseQueries("media_attach", false)
		m.metrics.RecordDatabaseQueryDuration("media_attach", time.Since(start))
		m.log.Warn("Post not found during media attach", slog.Int64("post_id", postID))
		return custom_errors.ErrPostNotFound
	}

	batch := &pgx.Batch{}
	for _, md := range media {
		batch.Queue(
			`INSERT INTO post_media (post_id, url, type, position) VALUES (@post_id, @url, @type, @position)`,
			pgx.NamedArgs{"post_id": postID, "url": md.URL, "type": md.Type, "position": md.Position},
		)
	}

	result := m.db.SendBatch(ctx, batch)
	defer func(result pgx.BatchResults) {
		err := result.Close()
		if err != nil {
			m.log.Error("Failed to close batch result in Attach media", slog.String("error", err.Error()), slog.Int64("post_id", postID))
		}
	}(result)

	if _, err := result.Exec(); err != nil {
		m.log.Error("Media attach failed", slog.String("error", err.Error()), slog.Int64("post_id", postID))
		m.metrics.IncrementDatabaseQueries("media_attach", false)
		m.metrics.RecordDatabaseQueryDuration("media_attach", time.Since(start))
		return custom_errors.ErrMediaAttachFailed
	}
	m.metrics.IncrementDatabaseQueries("media_attach", true)
	m.metrics.RecordDatabaseQueryDuration("media_attach", time.Since(start))
	return nil
}

func (m *MediaRepository) Reorder(ctx context.Context, postID int64, newPositions map[int64]int) error {
	start := time.Now()
	batch := &pgx.Batch{}
	for mediaID, position := range newPositions {
		batch.Queue(
			`UPDATE post_media SET position = @position WHERE post_id = @post_id AND id = @id`,
			pgx.NamedArgs{"position": position, "post_id": postID, "id": mediaID},
		)
	}

	result := m.db.SendBatch(ctx, batch)
	defer func(result pgx.BatchResults) {
		err := result.Close()
		if err != nil {
			m.log.Error("Failed to close batch result in Reorder media", slog.String("error", err.Error()), slog.Int64("post_id", postID))
		}
	}(result)

	if _, err := result.Exec(); err != nil {
		m.log.Error("Media reorder failed", slog.String("error", err.Error()), slog.Int64("post_id", postID))
		m.metrics.RecordDatabaseQueryDuration("media_reorder", time.Since(start))
		m.metrics.IncrementDatabaseQueries("media_reorder", false)
		return custom_errors.ErrMediaReorderFailed
	}
	m.metrics.RecordDatabaseQueryDuration("media_reorder", time.Since(start))
	m.metrics.IncrementDatabaseQueries("media_reorder", true)
	return nil
}

func (m *MediaRepository) Detach(ctx context.Context, mediaIDs []int64) error {
	start := time.Now()
	_, err := m.db.Exec(ctx, `DELETE FROM post_media WHERE id = ANY(@ids)`, pgx.NamedArgs{"ids": mediaIDs})
	if err != nil {
		m.log.Error("Media detach failed", slog.String("error", err.Error()), slog.Any("media_ids", mediaIDs))
		m.metrics.RecordDatabaseQueryDuration("media_detach", time.Since(start))
		m.metrics.IncrementDatabaseQueries("media_detach", false)
		return custom_errors.ErrMediaDetachFailed
	}
	m.metrics.RecordDatabaseQueryDuration("media_detach", time.Since(start))
	m.metrics.IncrementDatabaseQueries("media_detach", true)
	return nil
}

func (m *MediaRepository) GetByPost(ctx context.Context, postID int64) ([]*model.PostMedia, error) {
	start := time.Now()
	rows, err := m.db.Query(ctx, `SELECT id, url, type, position, created_at FROM post_media WHERE post_id = @postID ORDER BY position`, pgx.NamedArgs{"postID": postID})
	if err != nil {
		m.log.Error("Media query failed", slog.String("error", err.Error()), slog.Int64("post_id", postID))
		m.metrics.RecordDatabaseQueryDuration("media_get_by_post", time.Since(start))
		m.metrics.IncrementDatabaseQueries("media_get_by_post", false)
		return nil, custom_errors.ErrMediaQueryFailed
	}
	defer rows.Close()

	var media []*model.PostMedia
	for rows.Next() {
		var pm model.PostMedia
		if err := rows.Scan(&pm.ID, &pm.URL, &pm.Type, &pm.Position, &pm.CreatedAt); err != nil {
			m.metrics.RecordDatabaseQueryDuration("media_get_by_post", time.Since(start))
			m.metrics.IncrementDatabaseQueries("media_get_by_post", false)
			return nil, custom_errors.ErrDatabaseQuery
		}
		media = append(media, &pm)
	}
	m.log.Debug("Retrieved media for post", slog.Int64("post_id", postID), slog.Int("count", len(media)))
	m.metrics.RecordDatabaseQueryDuration("media_get_by_post", time.Since(start))
	m.metrics.IncrementDatabaseQueries("media_get_by_post", true)
	return media, nil
}

func (m *MediaRepository) GetByPosts(ctx context.Context, postIDs []int64) (map[int64][]*model.PostMedia, error) {
	start := time.Now()
	rows, err := m.db.Query(ctx, `SELECT post_id, id, url, type, position, created_at FROM post_media WHERE post_id = ANY(@postIDs) ORDER BY post_id, position`, pgx.NamedArgs{"postIDs": postIDs})
	if err != nil {
		m.log.Error("Batch media query failed", slog.String("error", err.Error()), slog.Any("post_ids", postIDs))
		m.metrics.RecordDatabaseQueryDuration("media_get_by_posts", time.Since(start))
		m.metrics.IncrementDatabaseQueries("media_get_by_posts", false)
		return nil, custom_errors.ErrMediaBatchQueryFailed
	}
	defer rows.Close()

	result := make(map[int64][]*model.PostMedia)
	var currentPostID int64 = -1
	var mediaGroup []*model.PostMedia

	for rows.Next() {
		var postID int64
		var pm model.PostMedia
		if err := rows.Scan(&postID, &pm.ID, &pm.URL, &pm.Type, &pm.Position, &pm.CreatedAt); err != nil {
			m.metrics.RecordDatabaseQueryDuration("media_get_by_posts", time.Since(start))
			m.metrics.IncrementDatabaseQueries("media_get_by_posts", false)
			return nil, custom_errors.ErrDatabaseQuery
		}

		if postID != currentPostID {
			if currentPostID != -1 {
				result[currentPostID] = mediaGroup
				mediaGroup = nil
			}
			currentPostID = postID
		}
		mediaGroup = append(mediaGroup, &pm)
	}

	if currentPostID != -1 {
		result[currentPostID] = mediaGroup
	}
	m.log.Debug("Retrieved media for batch posts", slog.Int("post_count", len(result)))
	m.metrics.RecordDatabaseQueryDuration("media_get_by_posts", time.Since(start))
	m.metrics.IncrementDatabaseQueries("media_get_by_posts", true)
	return result, nil
}
