package model

type PostMediaInput struct {
	URL  string    `json:"url"`
	Type MediaType `json:"type"`
}
