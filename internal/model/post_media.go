package model

import (
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostMedia struct {
	ID        int64            `json:"id"`
	PostID    int64            `json:"post_id"`
	URL       string           `json:"url"`
	Type      MediaType        `json:"type"`
	Position  int32            `json:"position"`
	CreatedAt pgtype.Timestamp `json:"created_at"`
}

type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"
)

func (t MediaType) IsValid() error {
	switch t {
	case MediaTypeImage, MediaTypeVideo:
		return nil
	}
	return fmt.Errorf("invalid media type: %s", t)
}

func (t *MediaType) UnmarshalText(text []byte) error {
	mt := MediaType(text)
	if err := mt.IsValid(); err != nil {
		return err
	}
	*t = mt
	return nil
}
