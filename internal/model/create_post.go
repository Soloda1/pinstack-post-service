package model

type CreatePostDTO struct {
	AuthorID   int64             `json:"author_id"`
	Title      string            `json:"title"`
	Content    *string           `json:"content,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	MediaItems []*PostMediaInput `json:"media_items,omitempty"`
}
