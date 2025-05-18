package model

type PostMediaInput struct {
	URL      string    `json:"url"`
	Type     MediaType `json:"type"`
	Position int32     `json:"position"`
}
