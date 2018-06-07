package example

import (
	"github.com/satori/go.uuid"
)

//go:generate goen -o goen.go
type Blog struct {
	BlogID uuid.UUID `goen:"" table:"blogs" primary_key:""`

	Name string

	Author string

	Posts []*Post `foreign_key:"blog_id"`
}

type Post struct {
	BlogID uuid.UUID `goen:"" table:"posts"`

	PostID int `primary_key:",omitempty"`

	Title string

	Content string

	Blog *Blog `foreign_key:"blog_id"`
}
