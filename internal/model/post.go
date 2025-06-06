package model

import "github.com/jackc/pgx/v5/pgtype"

type Post struct {
	ID        int64            `json:"id"`
	AuthorID  int64            `json:"author_id"`
	Title     string           `json:"title"`
	Content   *string          `json:"content,omitempty"`
	CreatedAt pgtype.Timestamp `json:"created_at"`
	UpdatedAt pgtype.Timestamp `json:"updated_at"`
}
