package example

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

//go:generate goen -o goen.go
type Blog struct {
	BlogID uuid.UUID `goen:"" table:"blogs" primary_key:""`

	Name string

	Author string

	Posts []*Post `foreign_key:"blog_id"`
}

type Post struct {
	Timestamp

	BlogID uuid.UUID `goen:"" table:"posts"`

	PostID int `primary_key:",omitempty"`

	Title string

	Content string

	// "order" is a sql keyword
	Order int

	Blog *Blog `foreign_key:"blog_id"`
}

type Timestamp struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
