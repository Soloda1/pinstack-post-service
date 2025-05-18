package model

type UpdatePostDTO struct {
	Title      *string           `json:"title,omitempty"`
	Content    *string           `json:"content,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	MediaItems []*PostMediaInput `json:"media_items,omitempty"`
}
