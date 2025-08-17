package tag_repository_postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	model "pinstack-post-service/internal/domain/models"
	ports "pinstack-post-service/internal/domain/ports/output"
	"pinstack-post-service/internal/infrastructure/outbound/repository/postgres/db"
	"time"

	"github.com/soloda1/pinstack-proto-definitions/custom_errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type TagRepository struct {
	log     ports.Logger
	db      db.PgDB
	metrics ports.MetricsProvider
}

func NewTagRepository(db db.PgDB, log ports.Logger, metrics ports.MetricsProvider) *TagRepository {
	return &TagRepository{db: db, log: log, metrics: metrics}
}

func (t *TagRepository) FindByNames(ctx context.Context, names []string) ([]*model.Tag, error) {
	start := time.Now()
	if len(names) == 0 {
		return nil, nil
	}

	query := `SELECT id, name FROM tags WHERE name = ANY(@names)`
	args := pgx.NamedArgs{"names": names}

	rows, err := t.db.Query(ctx, query, args)
	if err != nil {
		t.metrics.IncrementDatabaseQueries("tag_find_by_names", false)
		t.metrics.RecordDatabaseQueryDuration("tag_find_by_names", time.Since(start))
		t.log.Error("Error finding tags by names", slog.String("error", err.Error()))
		return nil, custom_errors.ErrTagQueryFailed
	}
	defer rows.Close()

	var tags []*model.Tag
	for rows.Next() {
		var tag model.Tag
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			t.log.Error("Error scanning tag row", slog.String("error", err.Error()))
			return nil, custom_errors.ErrTagScanFailed
		}
		tags = append(tags, &tag)
	}
	return tags, nil
}

func (t *TagRepository) FindByPost(ctx context.Context, postID int64) ([]*model.Tag, error) {
	start := time.Now()
	query := `
		SELECT t.id, t.name 
		FROM tags t
		INNER JOIN posts_tags pt ON pt.tag_id = t.id
		WHERE pt.post_id = @post_id`

	args := pgx.NamedArgs{"post_id": postID}

	rows, err := t.db.Query(ctx, query, args)
	if err != nil {
		t.metrics.IncrementDatabaseQueries("tag_find_by_post", false)
		t.metrics.RecordDatabaseQueryDuration("tag_find_by_post", time.Since(start))
		t.log.Error("Error finding tags by post", slog.Int64("post_id", postID), slog.String("error", err.Error()))
		return nil, custom_errors.ErrTagQueryFailed
	}
	defer rows.Close()

	var tags []*model.Tag
	for rows.Next() {
		var tag model.Tag
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			t.metrics.IncrementDatabaseQueries("tag_find_by_post", false)
			t.metrics.RecordDatabaseQueryDuration("tag_find_by_post", time.Since(start))
			t.log.Error("Error scanning tag row", slog.String("error", err.Error()))
			return nil, custom_errors.ErrTagScanFailed
		}
		tags = append(tags, &tag)
	}
	t.metrics.IncrementDatabaseQueries("tag_find_by_post", true)
	t.metrics.RecordDatabaseQueryDuration("tag_find_by_post", time.Since(start))
	return tags, nil
}

func (t *TagRepository) Create(ctx context.Context, name string) (*model.Tag, error) {
	start := time.Now()
	query := `
		INSERT INTO tags(name)
		VALUES (@name)
		ON CONFLICT (name) DO NOTHING
		RETURNING id, name`

	args := pgx.NamedArgs{"name": name}

	var tag model.Tag
	err := t.db.QueryRow(ctx, query, args).Scan(&tag.ID, &tag.Name)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			tags, findErr := t.FindByNames(ctx, []string{name})
			if findErr != nil || len(tags) == 0 {
				t.metrics.IncrementTagOperations("create", false)
				t.metrics.RecordDatabaseQueryDuration("tag_create", time.Since(start))
				t.log.Error("Tag exists but could not fetch", slog.String("name", name), slog.String("error", findErr.Error()))
				return nil, fmt.Errorf("failed to fetch existing tag: %w", findErr)
			}
			t.metrics.IncrementTagOperations("create", true)
			t.metrics.RecordDatabaseQueryDuration("tag_create", time.Since(start))
			return tags[0], nil
		}
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) && pgerr.Code == "23505" {
			tags, findErr := t.FindByNames(ctx, []string{name})
			if findErr != nil || len(tags) == 0 {
				t.metrics.IncrementTagOperations("create", false)
				t.metrics.RecordDatabaseQueryDuration("tag_create", time.Since(start))
				t.log.Error("Tag exists but could not fetch", slog.String("name", name), slog.String("error", findErr.Error()))
				return nil, fmt.Errorf("failed to fetch existing tag: %w", findErr)
			}
			t.metrics.IncrementTagOperations("create", true)
			t.metrics.RecordDatabaseQueryDuration("tag_create", time.Since(start))
			return tags[0], nil
		}
		t.metrics.IncrementTagOperations("create", false)
		t.metrics.RecordDatabaseQueryDuration("tag_create", time.Since(start))
		t.log.Error("Error creating tag", slog.String("name", name), slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}
	t.metrics.IncrementTagOperations("create", true)
	t.metrics.RecordDatabaseQueryDuration("tag_create", time.Since(start))
	return &tag, nil
}

func (t *TagRepository) DeleteUnused(ctx context.Context) error {
	start := time.Now()
	query := `DELETE FROM tags WHERE id NOT IN (SELECT DISTINCT tag_id FROM posts_tags)`

	_, err := t.db.Exec(ctx, query)
	if err != nil {
		t.log.Error("Error deleting unused tags", slog.String("error", err.Error()))
		t.metrics.RecordDatabaseQueryDuration("tag_delete_unused", time.Since(start))
		t.metrics.IncrementDatabaseQueries("tag_delete_unused", false)
		return custom_errors.ErrTagDeleteFailed
	}
	t.metrics.RecordDatabaseQueryDuration("tag_delete_unused", time.Since(start))
	t.metrics.IncrementDatabaseQueries("tag_delete_unused", true)
	return nil
}

func (t *TagRepository) TagPost(ctx context.Context, postID int64, tagNames []string) error {
	start := time.Now()
	if len(tagNames) == 0 {
		return nil
	}

	_, err := t.db.Exec(ctx, "SELECT 1 FROM posts WHERE id = @post_id", pgx.NamedArgs{"post_id": postID})
	if err != nil {
		t.metrics.IncrementTagOperations("tag_post", false)
		t.metrics.RecordDatabaseQueryDuration("tag_post", time.Since(start))
		if errors.Is(err, pgx.ErrNoRows) {
			return custom_errors.ErrPostNotFound
		}
		return custom_errors.ErrTagVerifyPostFailed
	}

	batch := &pgx.Batch{}
	query := `INSERT INTO posts_tags (post_id, tag_id) VALUES (@post_id, (SELECT id FROM tags WHERE name = @tag_name))`

	for _, tagName := range tagNames {
		args := pgx.NamedArgs{
			"post_id":  postID,
			"tag_name": tagName,
		}
		batch.Queue(query, args)
	}

	br := t.db.SendBatch(ctx, batch)
	defer func(br pgx.BatchResults) {
		err := br.Close()
		if err != nil {
			t.log.Error("Failed to close batch result in TagPost", slog.String("error", err.Error()), slog.Int64("post_id", postID))
		}
	}(br)

	for range tagNames {
		_, err := br.Exec()
		if err != nil {
			var pgerr *pgconn.PgError
			if errors.As(err, &pgerr) {
				switch pgerr.Code {
				case "23505":
					continue
				case "23503":
					t.metrics.IncrementTagOperations("tag_post", false)
					t.metrics.RecordDatabaseQueryDuration("tag_post", time.Since(start))
					return custom_errors.ErrTagNotFound
				}
			}
			t.metrics.IncrementTagOperations("tag_post", false)
			t.metrics.RecordDatabaseQueryDuration("tag_post", time.Since(start))
			t.log.Error("Error tagging post", slog.Int64("post_id", postID), slog.String("error", err.Error()))
			return custom_errors.ErrTagPost
		}
	}
	t.metrics.IncrementTagOperations("tag_post", true)
	t.metrics.RecordDatabaseQueryDuration("tag_post", time.Since(start))
	return nil
}

func (t *TagRepository) UntagPost(ctx context.Context, postID int64, tagNames []string) error {
	start := time.Now()
	if len(tagNames) == 0 {
		return nil
	}

	_, err := t.db.Exec(ctx, "SELECT 1 FROM posts WHERE id = @post_id", pgx.NamedArgs{"post_id": postID})
	if err != nil {
		t.metrics.IncrementTagOperations("untag_post", false)
		t.metrics.RecordDatabaseQueryDuration("untag_post", time.Since(start))
		if errors.Is(err, pgx.ErrNoRows) {
			return custom_errors.ErrPostNotFound
		}
		return custom_errors.ErrTagVerifyPostFailed
	}

	batch := &pgx.Batch{}
	query := `DELETE FROM posts_tags 
		WHERE post_id = @post_id 
		AND tag_id = (SELECT id FROM tags WHERE name = @tag_name)`

	for _, tagName := range tagNames {
		args := pgx.NamedArgs{
			"post_id":  postID,
			"tag_name": tagName,
		}
		batch.Queue(query, args)
	}

	br := t.db.SendBatch(ctx, batch)
	defer func(br pgx.BatchResults) {
		err := br.Close()
		if err != nil {
			t.log.Error("Failed to close batch result in UntagPost", slog.String("error", err.Error()), slog.Int64("post_id", postID))
		}
	}(br)

	for range tagNames {
		_, err := br.Exec()
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			var pgerr *pgconn.PgError
			if errors.As(err, &pgerr) && pgerr.Code == "23503" {
				t.metrics.IncrementTagOperations("untag_post", false)
				t.metrics.RecordDatabaseQueryDuration("untag_post", time.Since(start))
				return custom_errors.ErrTagNotFound
			}
			t.metrics.IncrementTagOperations("untag_post", false)
			t.metrics.RecordDatabaseQueryDuration("untag_post", time.Since(start))
			t.log.Error("Error untagging post", slog.Int64("post_id", postID), slog.String("error", err.Error()))
			return err
		}
	}
	t.metrics.IncrementTagOperations("untag_post", true)
	t.metrics.RecordDatabaseQueryDuration("untag_post", time.Since(start))
	return nil
}

func (t *TagRepository) ReplacePostTags(ctx context.Context, postID int64, newTags []string) error {
	start := time.Now()
	_, err := t.db.Exec(ctx, "SELECT 1 FROM posts WHERE id = @post_id", pgx.NamedArgs{"post_id": postID})
	if err != nil {
		t.metrics.IncrementTagOperations("replace_post_tags", false)
		t.metrics.RecordDatabaseQueryDuration("replace_post_tags", time.Since(start))
		if errors.Is(err, pgx.ErrNoRows) {
			return custom_errors.ErrPostNotFound
		}
		return fmt.Errorf("failed to verify post: %w", err)
	}

	deleteQuery := `DELETE FROM posts_tags WHERE post_id = @post_id`
	_, err = t.db.Exec(ctx, deleteQuery, pgx.NamedArgs{"post_id": postID})
	if err != nil {
		t.metrics.IncrementTagOperations("replace_post_tags", false)
		t.metrics.RecordDatabaseQueryDuration("replace_post_tags", time.Since(start))
		t.log.Error("Error deleting old tags", slog.String("error", err.Error()))
		return custom_errors.ErrDatabaseQuery
	}

	if len(newTags) > 0 {
		batch := &pgx.Batch{}
		insertQuery := `INSERT INTO posts_tags (post_id, tag_id) VALUES (@post_id, (SELECT id FROM tags WHERE name = @tag_name))`

		for _, tagName := range newTags {
			batch.Queue(insertQuery, pgx.NamedArgs{
				"post_id":  postID,
				"tag_name": tagName,
			})
		}

		br := t.db.SendBatch(ctx, batch)
		defer func(br pgx.BatchResults) {
			err := br.Close()
			if err != nil {
				t.log.Error("Failed to close batch result in ReplacePostTags", slog.String("error", err.Error()), slog.Int64("post_id", postID))
			}
		}(br)

		for range newTags {
			_, err := br.Exec()
			if err != nil {
				var pgerr *pgconn.PgError
				if errors.As(err, &pgerr) && pgerr.Code == "23503" {
					t.metrics.IncrementTagOperations("replace_post_tags", false)
					t.metrics.RecordDatabaseQueryDuration("replace_post_tags", time.Since(start))
					return custom_errors.ErrTagNotFound
				}
				t.metrics.IncrementTagOperations("replace_post_tags", false)
				t.metrics.RecordDatabaseQueryDuration("replace_post_tags", time.Since(start))
				t.log.Error("Error inserting new tags", slog.Int64("post_id", postID), slog.String("error", err.Error()))
				return custom_errors.ErrDatabaseQuery
			}
		}
	}

	t.metrics.IncrementTagOperations("replace_post_tags", true)
	t.metrics.RecordDatabaseQueryDuration("replace_post_tags", time.Since(start))
	return nil
}
