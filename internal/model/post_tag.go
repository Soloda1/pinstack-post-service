package model

import "github.com/jackc/pgx/v5/pgtype"

type PostTag struct {
	ID        int64            `json:"id"`
	PostID    int64            `json:"post_id"`
	Tag       string           `json:"tag"`
	CreatedAt pgtype.Timestamp `json:"created_at"`
}
