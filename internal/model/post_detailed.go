package model

type PostDetailed struct {
	Post   *Post        `json:"post,omitempty"`
	Author *User        `json:"author,omitempty"` // Добавлено поле для информации об авторе
	Media  *[]PostMedia `json:"media,omitempty"`
	Tags   *[]PostTag   `json:"tags,omitempty"`
}
