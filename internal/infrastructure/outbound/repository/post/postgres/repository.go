package post_repository_postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	ports "pinstack-post-service/internal/domain/ports/output"
	"pinstack-post-service/internal/infrastructure/outbound/repository/postgres/db"
	"strings"
	"time"

	"github.com/soloda1/pinstack-proto-definitions/custom_errors"

	model "pinstack-post-service/internal/domain/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostRepository struct {
	log     ports.Logger
	db      db.PgDB
	metrics ports.MetricsProvider
}

func NewPostRepository(db db.PgDB, log ports.Logger, metrics ports.MetricsProvider) *PostRepository {
	return &PostRepository{db: db, log: log, metrics: metrics}
}

func (p *PostRepository) Create(ctx context.Context, post *model.Post) (*model.Post, error) {
	start := time.Now()
	p.log.Debug("Creating new post", slog.Int64("author_id", post.AuthorID), slog.String("title", post.Title))

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
		p.metrics.IncrementDatabaseQueries("post_create", false)
		p.metrics.RecordDatabaseQueryDuration("post_create", time.Since(start))
		p.log.Error("Error creating post", slog.String("error", err.Error()))
		return nil, custom_errors.ErrDatabaseQuery
	}

	p.metrics.IncrementDatabaseQueries("post_create", true)
	p.metrics.RecordDatabaseQueryDuration("post_create", time.Since(start))
	p.log.Debug("Successfully created post", slog.Int64("id", createdPost.ID), slog.Int64("author_id", createdPost.AuthorID))
	return &createdPost, nil
}

func (p *PostRepository) GetByID(ctx context.Context, id int64) (*model.Post, error) {
	start := time.Now()
	p.log.Debug("Getting post by ID", slog.Int64("id", id))

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
			p.metrics.IncrementDatabaseQueries("post_get_by_id", false)
			p.metrics.RecordDatabaseQueryDuration("post_get_by_id", time.Since(start))
			p.log.Debug("Post not found by id", slog.Int64("id", id), slog.String("error", err.Error()))
			return nil, custom_errors.ErrPostNotFound
		}
		p.metrics.IncrementDatabaseQueries("post_get_by_id", false)
		p.metrics.RecordDatabaseQueryDuration("post_get_by_id", time.Since(start))
		p.log.Error("Error getting post by id", slog.Int64("id", id), slog.String("error", err.Error()))
		return nil, custom_errors.ErrDatabaseQuery
	}
	p.metrics.IncrementDatabaseQueries("post_get_by_id", true)
	p.metrics.RecordDatabaseQueryDuration("post_get_by_id", time.Since(start))
	p.log.Debug("Successfully retrieved post by ID", slog.Int64("id", post.ID), slog.Int64("author_id", post.AuthorID))
	return post, nil
}

func (p *PostRepository) GetByAuthor(ctx context.Context, authorID int64) ([]*model.Post, error) {
	start := time.Now()
	p.log.Debug("Getting posts by author", slog.Int64("author_id", authorID))

	args := pgx.NamedArgs{"author_id": authorID}
	query := `SELECT id, author_id, title, content, created_at, updated_at
				FROM posts WHERE author_id = @author_id ORDER BY created_at DESC`

	rows, err := p.db.Query(ctx, query, args)
	if err != nil {
		p.metrics.IncrementDatabaseQueries("post_get_by_author", false)
		p.metrics.RecordDatabaseQueryDuration("post_get_by_author", time.Since(start))
		p.log.Error("Error getting posts by author", slog.Int64("author_id", authorID), slog.String("error", err.Error()))
		return nil, custom_errors.ErrDatabaseQuery
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
			p.metrics.IncrementDatabaseQueries("post_get_by_author", false)
			p.metrics.RecordDatabaseQueryDuration("post_get_by_author", time.Since(start))
			p.log.Error("Error scanning post during GetByAuthor", slog.Int64("author_id", authorID), slog.String("error", err.Error()))
			return nil, custom_errors.ErrDatabaseQuery
		}
		posts = append(posts, &post)
	}

	if err = rows.Err(); err != nil {
		p.metrics.IncrementDatabaseQueries("post_get_by_author", false)
		p.metrics.RecordDatabaseQueryDuration("post_get_by_author", time.Since(start))
		p.log.Error("Error iterating rows during GetByAuthor", slog.Int64("author_id", authorID), slog.String("error", err.Error()))
		return nil, custom_errors.ErrDatabaseQuery
	}

	p.metrics.IncrementDatabaseQueries("post_get_by_author", true)
	p.metrics.RecordDatabaseQueryDuration("post_get_by_author", time.Since(start))
	p.log.Debug("Successfully retrieved posts by author", slog.Int64("author_id", authorID), slog.Int("count", len(posts)))
	return posts, nil
}

func (p *PostRepository) Update(ctx context.Context, id int64, update *model.UpdatePostDTO) (*model.Post, error) {
	start := time.Now()
	p.log.Debug("Updating post", slog.Int64("id", id), slog.Any("update_fields", map[string]bool{
		"title":   update.Title != nil,
		"content": update.Content != nil,
	}))

	setClauses := []string{}
	args := pgx.NamedArgs{"id": id}

	if update.Title != nil && *update.Title != "" {
		setClauses = append(setClauses, "title = @title")
		args["title"] = *update.Title
		p.log.Debug("Updating post title", slog.Int64("id", id), slog.String("new_title", *update.Title))
	}
	if update.Content != nil && *update.Content != "" {
		setClauses = append(setClauses, "content = @content")
		args["content"] = *update.Content
		p.log.Debug("Updating post content", slog.Int64("id", id))
	}

	updatedAt := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	setClauses = append(setClauses, "updated_at = @updated_at")
	args["updated_at"] = updatedAt

	if len(setClauses) == 0 {
		p.metrics.IncrementDatabaseQueries("post_update", false)
		p.metrics.RecordDatabaseQueryDuration("post_update", time.Since(start))
		p.log.Debug("No fields to update", slog.Int64("id", id))
		return nil, custom_errors.ErrNoUpdateRows
	}

	p.log.Debug("Building update query", slog.Int64("id", id), slog.Int("set_clauses_count", len(setClauses)))
	query := "UPDATE posts SET " + strings.Join(setClauses, ", ") + " WHERE id = @id RETURNING id, author_id, title, content, created_at, updated_at"

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
			p.metrics.IncrementDatabaseQueries("post_update", false)
			p.metrics.RecordDatabaseQueryDuration("post_update", time.Since(start))
			p.log.Debug("Post not found by id during Update", slog.Int64("id", id), slog.String("error", err.Error()))
			return nil, custom_errors.ErrPostNotFound
		}
		p.metrics.IncrementDatabaseQueries("post_update", false)
		p.metrics.RecordDatabaseQueryDuration("post_update", time.Since(start))
		p.log.Error("Error updating post", slog.Int64("id", id), slog.String("error", err.Error()))
		return nil, custom_errors.ErrDatabaseQuery
	}

	p.metrics.IncrementDatabaseQueries("post_update", true)
	p.metrics.RecordDatabaseQueryDuration("post_update", time.Since(start))
	p.log.Debug("Successfully updated post", slog.Int64("id", updatedPost.ID), slog.Int64("author_id", updatedPost.AuthorID),
		slog.Time("updated_at", updatedPost.UpdatedAt.Time))
	return &updatedPost, nil
}

func (p *PostRepository) Delete(ctx context.Context, id int64) error {
	start := time.Now()
	p.log.Debug("Deleting post", slog.Int64("id", id))
	args := pgx.NamedArgs{"id": id}
	query := `DELETE FROM posts WHERE id = @id`
	result, err := p.db.Exec(ctx, query, args)
	if err != nil {
		p.metrics.IncrementDatabaseQueries("post_delete", false)
		p.metrics.RecordDatabaseQueryDuration("post_delete", time.Since(start))
		p.log.Error("Error deleting post", slog.Int64("id", id), slog.String("error", err.Error()))
		return custom_errors.ErrDatabaseQuery
	}
	if result.RowsAffected() == 0 {
		p.metrics.IncrementDatabaseQueries("post_delete", false)
		p.metrics.RecordDatabaseQueryDuration("post_delete", time.Since(start))
		p.log.Debug("Post not found during deletion", slog.Int64("id", id))
		return custom_errors.ErrPostNotFound
	}
	p.metrics.IncrementDatabaseQueries("post_delete", true)
	p.metrics.RecordDatabaseQueryDuration("post_delete", time.Since(start))
	p.log.Debug("Successfully deleted post", slog.Int64("id", id))
	return nil
}

func (p *PostRepository) List(ctx context.Context, filters model.PostFilters) ([]*model.Post, int, error) {
	start := time.Now()
	p.log.Debug("Listing posts with filters",
		slog.Any("author_id", filters.AuthorID),
		slog.Any("created_after", filters.CreatedAfter),
		slog.Any("created_before", filters.CreatedBefore),
		slog.Any("tag_names", filters.TagNames),
		slog.Any("limit", filters.Limit),
		slog.Any("offset", filters.Offset))

	args := pgx.NamedArgs{}
	baseQuery := `SELECT DISTINCT p.id, p.author_id, p.title, p.content, p.created_at, p.updated_at FROM posts p`

	whereClauses := []string{}

	if filters.AuthorID != nil {
		whereClauses = append(whereClauses, "p.author_id = @author_id")
		args["author_id"] = *filters.AuthorID
		p.log.Debug("Adding author filter", slog.Int64("author_id", *filters.AuthorID))
	}
	if filters.CreatedAfter != nil {
		whereClauses = append(whereClauses, "p.created_at > @created_after")
		args["created_after"] = *filters.CreatedAfter
		p.log.Debug("Adding created_after filter", slog.Any("created_after", filters.CreatedAfter), slog.String("operator", ">"))
	}
	if filters.CreatedBefore != nil {
		whereClauses = append(whereClauses, "p.created_at < @created_before")
		args["created_before"] = *filters.CreatedBefore
		p.log.Debug("Adding created_before filter", slog.Any("created_before", filters.CreatedBefore), slog.String("operator", "<"))
	}

	if len(filters.TagNames) > 0 {
		p.log.Debug("Adding tags filter", slog.Any("tag_names", filters.TagNames))
		baseQuery += ` JOIN posts_tags pt ON p.id = pt.post_id JOIN tags t ON pt.tag_id = t.id`
		var tagClauses []string
		for i, tagName := range filters.TagNames {
			paramName := fmt.Sprintf("tag_name_%d", i)
			tagClauses = append(tagClauses, fmt.Sprintf("t.name ILIKE @%s", paramName))
			args[paramName] = tagName
			p.log.Debug("Adding tag filter", slog.String("tag_name", tagName), slog.String("param_name", paramName))
		}
		whereClauses = append(whereClauses, "("+strings.Join(tagClauses, " OR ")+")")
	}

	if len(whereClauses) > 0 {
		condition := " WHERE " + strings.Join(whereClauses, " AND ")
		baseQuery += condition
		p.log.Debug("Added WHERE conditions", slog.Int("conditions_count", len(whereClauses)))
	}

	baseQuery += " ORDER BY p.created_at DESC"
	p.log.Debug("Query before pagination", slog.String("query", baseQuery))

	if filters.Limit != nil {
		baseQuery += " LIMIT @limit"
		args["limit"] = *filters.Limit
		p.log.Debug("Adding pagination limit", slog.Int("limit", *filters.Limit))
	}
	if filters.Offset != nil {
		baseQuery += " OFFSET @offset"
		args["offset"] = *filters.Offset
		p.log.Debug("Adding pagination offset", slog.Int("offset", *filters.Offset))
	}

	p.log.Debug("Executing list query", slog.String("query", baseQuery), slog.Any("args_keys", args))
	rows, err := p.db.Query(ctx, baseQuery, args)
	if err != nil {
		p.log.Error("Error listing posts", slog.String("error", err.Error()))
		return nil, 0, custom_errors.ErrDatabaseQuery
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
			return nil, 0, custom_errors.ErrDatabaseQuery
		}
		posts = append(posts, &post)
		p.log.Debug("Scanned post in List", slog.Int64("post_id", post.ID), slog.Int64("author_id", post.AuthorID))
	}

	if err = rows.Err(); err != nil {
		p.log.Error("Error iterating rows during List", slog.String("error", err.Error()))
		return nil, 0, custom_errors.ErrDatabaseQuery
	}

	p.log.Debug("Retrieved posts in List", slog.Int("retrieved_posts_count", len(posts)))

	p.log.Debug("Building count query")
	countQuery := "SELECT COUNT(DISTINCT p.id) FROM posts p"

	if len(filters.TagNames) > 0 {
		countQuery += " JOIN posts_tags pt ON p.id = pt.post_id JOIN tags t ON pt.tag_id = t.id"
	}

	if len(whereClauses) > 0 {
		countQuery += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Create a copy of args without LIMIT and OFFSET for the count query
	countArgs := make(pgx.NamedArgs)
	for k, v := range args {
		if k != "limit" && k != "offset" {
			countArgs[k] = v
		}
	}

	var total int
	p.log.Debug("Executing count query", slog.String("count_query", countQuery), slog.Any("args_keys", countArgs))
	err = p.db.QueryRow(ctx, countQuery, countArgs).Scan(&total)
	if err != nil {
		p.metrics.IncrementDatabaseQueries("post_list", false)
		p.metrics.RecordDatabaseQueryDuration("post_list", time.Since(start))
		p.log.Error("Error counting posts", slog.String("error", err.Error()))
		return nil, 0, custom_errors.ErrDatabaseQuery
	}
	p.log.Debug("Count query result", slog.Int("total", total))

	p.metrics.IncrementDatabaseQueries("post_list", true)
	p.metrics.RecordDatabaseQueryDuration("post_list", time.Since(start))
	return posts, total, nil
}
