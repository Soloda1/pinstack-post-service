package model

type PostMediaInput struct {
	URL  string `json:"url"`
	Type string `json:"type"` // "image" | "video"
}
