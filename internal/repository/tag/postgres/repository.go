package tag_repository_postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/soloda1/pinstack-proto-definitions/custom_errors"
	"log/slog"
	"pinstack-post-service/internal/logger"
	"pinstack-post-service/internal/model"
	"pinstack-post-service/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type TagRepository struct {
	log *logger.Logger
	db  db.PgDB
}

func NewTagRepository(db db.PgDB, log *logger.Logger) *TagRepository {
	return &TagRepository{db: db, log: log}
}

func (t *TagRepository) FindByNames(ctx context.Context, names []string) ([]*model.Tag, error) {
	if len(names) == 0 {
		return nil, nil
	}

	query := `SELECT id, name FROM tags WHERE name = ANY(@names)`
	args := pgx.NamedArgs{"names": names}

	rows, err := t.db.Query(ctx, query, args)
	if err != nil {
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
	query := `
		SELECT t.id, t.name 
		FROM tags t
		INNER JOIN posts_tags pt ON pt.tag_id = t.id
		WHERE pt.post_id = @post_id`

	args := pgx.NamedArgs{"post_id": postID}

	rows, err := t.db.Query(ctx, query, args)
	if err != nil {
		t.log.Error("Error finding tags by post", slog.Int64("post_id", postID), slog.String("error", err.Error()))
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

func (t *TagRepository) Create(ctx context.Context, name string) (*model.Tag, error) {
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
				t.log.Error("Tag exists but could not fetch", slog.String("name", name), slog.String("error", findErr.Error()))
				return nil, fmt.Errorf("failed to fetch existing tag: %w", findErr)
			}
			return tags[0], nil
		}
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) && pgerr.Code == "23505" {
			tags, findErr := t.FindByNames(ctx, []string{name})
			if findErr != nil || len(tags) == 0 {
				t.log.Error("Tag exists but could not fetch", slog.String("name", name), slog.String("error", findErr.Error()))
				return nil, fmt.Errorf("failed to fetch existing tag: %w", findErr)
			}
			return tags[0], nil
		}
		t.log.Error("Error creating tag", slog.String("name", name), slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}
	return &tag, nil
}

func (t *TagRepository) DeleteUnused(ctx context.Context) error {
	query := `DELETE FROM tags WHERE id NOT IN (SELECT DISTINCT tag_id FROM posts_tags)`

	_, err := t.db.Exec(ctx, query)
	if err != nil {
		t.log.Error("Error deleting unused tags", slog.String("error", err.Error()))
		return custom_errors.ErrTagDeleteFailed
	}
	return nil
}

func (t *TagRepository) TagPost(ctx context.Context, postID int64, tagNames []string) error {
	if len(tagNames) == 0 {
		return nil
	}

	_, err := t.db.Exec(ctx, "SELECT 1 FROM posts WHERE id = @post_id", pgx.NamedArgs{"post_id": postID})
	if err != nil {
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
					return custom_errors.ErrTagNotFound
				}
			}
			t.log.Error("Error tagging post", slog.Int64("post_id", postID), slog.String("error", err.Error()))
			return custom_errors.ErrTagPost
		}
	}
	return nil
}

func (t *TagRepository) UntagPost(ctx context.Context, postID int64, tagNames []string) error {
	if len(tagNames) == 0 {
		return nil
	}

	_, err := t.db.Exec(ctx, "SELECT 1 FROM posts WHERE id = @post_id", pgx.NamedArgs{"post_id": postID})
	if err != nil {
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
				return custom_errors.ErrTagNotFound
			}
			t.log.Error("Error untagging post", slog.Int64("post_id", postID), slog.String("error", err.Error()))
			return err
		}
	}
	return nil
}

func (t *TagRepository) ReplacePostTags(ctx context.Context, postID int64, newTags []string) error {
	_, err := t.db.Exec(ctx, "SELECT 1 FROM posts WHERE id = @post_id", pgx.NamedArgs{"post_id": postID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return custom_errors.ErrPostNotFound
		}
		return fmt.Errorf("failed to verify post: %w", err)
	}

	deleteQuery := `DELETE FROM posts_tags WHERE post_id = @post_id`
	_, err = t.db.Exec(ctx, deleteQuery, pgx.NamedArgs{"post_id": postID})
	if err != nil {
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
					return custom_errors.ErrTagNotFound
				}
				t.log.Error("Error inserting new tags", slog.Int64("post_id", postID), slog.String("error", err.Error()))
				return custom_errors.ErrDatabaseQuery
			}
		}
	}

	return nil
}
