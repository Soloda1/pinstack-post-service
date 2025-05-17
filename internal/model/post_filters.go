package model

import "github.com/jackc/pgx/v5/pgtype"

type PostFilters struct {
	AuthorID      *int64
	TagNames      []string
	CreatedAfter  *pgtype.Timestamptz
	CreatedBefore *pgtype.Timestamptz
	Limit         *int
	Offset        *int
}
