package model

type UpdatePostDTO struct {
	UserID     int64             `json:"user_id"`
	Title      *string           `json:"title,omitempty"`
	Content    *string           `json:"content,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	MediaItems []*PostMediaInput `json:"media_items,omitempty"`
}
