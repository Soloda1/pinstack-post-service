package post_repository_postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"pinstack-post-service/internal/custom_errors"
	"pinstack-post-service/internal/logger"
	"pinstack-post-service/internal/model"
	"pinstack-post-service/internal/repository/postgres"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostRepository struct {
	log *logger.Logger
	db  postgres.PgDB
}

func NewPostRepository(db postgres.PgDB, log *logger.Logger) *PostRepository {
	return &PostRepository{db: db, log: log}
}

func (p *PostRepository) Create(ctx context.Context, post *model.Post) (*model.Post, error) {
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}

	args := pgx.NamedArgs{
		"author_id":  post.AuthorID,
		"title":      post.Title,
		"content":    post.Content,
		"created_at": now,
		"updated_at": now,
	}

	query := `
		INSERT INTO posts (author_id, title, content, created_at, updated_at)
		VALUES (@author_id, @title, @content, @created_at, @updated_at)
		RETURNING id, author_id, title, content, created_at, updated_at`

	var createdPost model.Post
	err := p.db.QueryRow(ctx, query, args).Scan(
		&createdPost.ID,
		&createdPost.AuthorID,
		&createdPost.Title,
		&createdPost.Content,
		&createdPost.CreatedAt,
		&createdPost.UpdatedAt,
	)

	if err != nil {
		p.log.Error("Error creating post", slog.String("error", err.Error()))
		return nil, err
	}

	return &createdPost, nil
}

func (p *PostRepository) GetByID(ctx context.Context, id int64) (*model.Post, error) {
	args := pgx.NamedArgs{"id": id}
	query := `SELECT id, author_id, title, content, created_at, updated_at
				FROM posts WHERE id = @id`
	row := p.db.QueryRow(ctx, query, args)
	post := &model.Post{}
	err := row.Scan(
		&post.ID,
		&post.AuthorID,
		&post.Title,
		&post.Content,
		&post.CreatedAt,
		&post.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			p.log.Debug("Post not found by id", slog.Int64("id", id), slog.String("error", err.Error()))
			return nil, custom_errors.ErrPostNotFound
		}
		p.log.Error("Error getting post by id", slog.Int64("id", id), slog.String("error", err.Error()))
		return nil, err
	}
	return post, nil
}

func (p *PostRepository) GetByAuthor(ctx context.Context, authorID int64) ([]*model.Post, error) {
	args := pgx.NamedArgs{"author_id": authorID}
	query := `SELECT id, author_id, title, content, created_at, updated_at
				FROM posts WHERE author_id = @author_id ORDER BY created_at DESC`

	rows, err := p.db.Query(ctx, query, args)
	if err != nil {
		p.log.Error("Error getting posts by author", slog.Int64("author_id", authorID), slog.String("error", err.Error()))
		return nil, err
	}
	defer rows.Close()

	var posts []*model.Post
	for rows.Next() {
		var post model.Post
		err := rows.Scan(
			&post.ID,
			&post.AuthorID,
			&post.Title,
			&post.Content,
			&post.CreatedAt,
			&post.UpdatedAt,
		)
		if err != nil {
			p.log.Error("Error scanning post during GetByAuthor", slog.Int64("author_id", authorID), slog.String("error", err.Error()))
			return nil, err
		}
		posts = append(posts, &post)
	}

	if err = rows.Err(); err != nil {
		p.log.Error("Error iterating rows during GetByAuthor", slog.Int64("author_id", authorID), slog.String("error", err.Error()))
		return nil, err
	}

	return posts, nil
}

func (p *PostRepository) UpdateTitle(ctx context.Context, id int64, title string) (*model.Post, error) {
	updatedAt := pgtype.Timestamp{Time: time.Now(), Valid: true}

	args := pgx.NamedArgs{
		"id":         id,
		"title":      title,
		"updated_at": updatedAt,
	}

	query := `UPDATE posts SET title = @title, updated_at = @updated_at WHERE id = @id
		RETURNING id, author_id, title, content, created_at, updated_at`

	var updatedPost model.Post
	err := p.db.QueryRow(ctx, query, args).Scan(
		&updatedPost.ID,
		&updatedPost.AuthorID,
		&updatedPost.Title,
		&updatedPost.Content,
		&updatedPost.CreatedAt,
		&updatedPost.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			p.log.Debug("Post not found by id during UpdateTitle", slog.Int64("id", id), slog.String("error", err.Error()))
			return nil, custom_errors.ErrPostNotFound
		}
		p.log.Error("Error updating post title", slog.Int64("id", id), slog.String("error", err.Error()))
		return nil, err
	}

	return &updatedPost, nil
}

func (p *PostRepository) UpdateContent(ctx context.Context, id int64, content string) (*model.Post, error) {
	updatedAt := pgtype.Timestamp{Time: time.Now(), Valid: true}

	args := pgx.NamedArgs{
		"id":         id,
		"content":    content,
		"updated_at": updatedAt,
	}

	query := `UPDATE posts SET content = @content, updated_at = @updated_at WHERE id = @id
		RETURNING id, author_id, title, content, created_at, updated_at`

	var updatedPost model.Post
	err := p.db.QueryRow(ctx, query, args).Scan(
		&updatedPost.ID,
		&updatedPost.AuthorID,
		&updatedPost.Title,
		&updatedPost.Content,
		&updatedPost.CreatedAt,
		&updatedPost.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			p.log.Debug("Post not found by id during UpdateContent", slog.Int64("id", id), slog.String("error", err.Error()))
			return nil, custom_errors.ErrPostNotFound
		}
		p.log.Error("Error updating post content", slog.Int64("id", id), slog.String("error", err.Error()))
		return nil, err
	}

	return &updatedPost, nil
}

func (p *PostRepository) Delete(ctx context.Context, id int64) error {
	args := pgx.NamedArgs{"id": id}
	query := `DELETE FROM posts WHERE id = @id`
	result, err := p.db.Exec(ctx, query, args)
	if err != nil {
		p.log.Error("Error deleting post", slog.Int64("id", id), slog.String("error", err.Error()))
		return err
	}
	if result.RowsAffected() == 0 {
		return custom_errors.ErrPostNotFound
	}
	return nil
}

func (p *PostRepository) List(ctx context.Context, filters model.PostFilters) ([]*model.Post, error) {
	args := pgx.NamedArgs{}
	baseQuery := `SELECT p.id, p.author_id, p.title, p.content, p.created_at, p.updated_at FROM posts p`

	whereClauses := []string{}

	if filters.AuthorID != nil {
		whereClauses = append(whereClauses, "p.author_id = @author_id")
		args["author_id"] = *filters.AuthorID
	}
	if filters.CreatedAfter != nil {
		// Assuming filters.CreatedAfter is pgtype.Timestamptz, convert to pgtype.Timestamp if posts table uses Timestamp
		// For simplicity, let's assume direct compatibility or that the types match (e.g. both are Timestamptz or Timestamp)
		whereClauses = append(whereClauses, "p.created_at >= @created_after")
		args["created_after"] = *filters.CreatedAfter
	}
	if filters.CreatedBefore != nil {
		whereClauses = append(whereClauses, "p.created_at <= @created_before")
		args["created_before"] = *filters.CreatedBefore
	}

	// Handling TagNames requires a JOIN with post_tags and tags tables.
	// Example: JOIN post_tags pt ON p.id = pt.post_id JOIN tags t ON pt.tag_id = t.id
	// AND t.name = ANY(@tag_names)
	if len(filters.TagNames) > 0 {
		baseQuery += ` JOIN post_tags pt ON p.id = pt.post_id JOIN tags t ON pt.tag_id = t.id`
		// Using = ANY for array comparison. Ensure TagNames is passed as a pgx array type if needed, or build OR clauses.
		// For simplicity with NamedArgs and ILIKE, let's build OR clauses for tags if there are few.
		// If many tags, consider passing as array: `t.name = ANY(@tag_names)` and args["tag_names"] = filters.TagNames
		var tagClauses []string
		for i, tagName := range filters.TagNames {
			paramName := fmt.Sprintf("tag_name_%d", i)
			tagClauses = append(tagClauses, fmt.Sprintf("t.name ILIKE @%s", paramName))
			args[paramName] = tagName
		}
		whereClauses = append(whereClauses, "("+strings.Join(tagClauses, " OR ")+")")
	}

	if len(whereClauses) > 0 {
		condition := " WHERE " + strings.Join(whereClauses, " AND ")
		baseQuery += condition
	}

	baseQuery += " ORDER BY p.created_at DESC"

	if filters.Limit != nil {
		baseQuery += " LIMIT @limit"
		args["limit"] = *filters.Limit
	}
	if filters.Offset != nil {
		baseQuery += " OFFSET @offset"
		args["offset"] = *filters.Offset
	}

	rows, err := p.db.Query(ctx, baseQuery, args)
	if err != nil {
		p.log.Error("Error listing posts", slog.String("error", err.Error()))
		return nil, err
	}
	defer rows.Close()

	var posts []*model.Post
	for rows.Next() {
		var post model.Post
		err := rows.Scan(
			&post.ID,
			&post.AuthorID,
			&post.Title,
			&post.Content,
			&post.CreatedAt,
			&post.UpdatedAt,
		)
		if err != nil {
			p.log.Error("Error scanning post during List", slog.String("error", err.Error()))
			return nil, err
		}
		posts = append(posts, &post)
	}

	if err = rows.Err(); err != nil {
		p.log.Error("Error iterating rows during List", slog.String("error", err.Error()))
		return nil, err
	}

	return posts, nil
}
