package model

type PostDetailed struct {
	Post   *Post        `json:"post,omitempty"`
	Author *User        `json:"author,omitempty"`
	Media  *[]PostMedia `json:"media,omitempty"`
	Tags   *[]PostTag   `json:"tags,omitempty"`
}
