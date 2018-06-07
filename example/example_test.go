package example

import (
	"database/sql"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	_ "github.com/kamichidu/goen/dialect/sqlite3"
	_ "github.com/mattn/go-sqlite3"
	"github.com/satori/go.uuid"
)

const ddl = `
drop table if exists blogs;
create table blogs (
	blog_id blob primary key,
	name varchar,
	author varchar(32)
);

drop table if exists posts;
create table posts (
	blog_id blob not null,
	post_id integer not null primary key,
	title varchar,
	content varchar,
	-- primary key(blog_id, post_id),
	foreign key (blog_id) references blogs(blog_id)
);
`

func Example() {
	db, err := sql.Open("sqlite3", "./sqlite.db")
	if err != nil {
		panic(err)
	}
	if _, err := db.Exec(ddl); err != nil {
		panic(err)
	}

	dbc := NewDBContext("sqlite3", db)
	dbc.DebugMode(true)

	src := []*Blog{
		&Blog{
			BlogID: uuid.Must(uuid.FromString("d03bc237-eef4-4b6f-afe1-ea901357d828")),
			Name:   "testing1",
			Author: "kamichidu",
		},
		&Blog{
			BlogID: uuid.Must(uuid.FromString("b95e5d4d-7eb9-4612-882d-224daa4a59ee")),
			Name:   "testing2",
			Author: "kamichidu",
		},
		&Blog{
			BlogID: uuid.Must(uuid.FromString("22f931c8-ac87-4520-88e8-83fc0604b8f5")),
			Name:   "testing3",
			Author: "kamichidu",
		},
		&Blog{
			BlogID: uuid.Must(uuid.FromString("065c6554-9aff-4b42-ab3b-141ed5ef5624")),
			Name:   "testing4",
			Author: "kamichidu",
		},
	}
	for _, blog := range src {
		dbc.Blog.Insert(blog)
	}
	func(blog *Blog) {
		dbc.Post.Insert(&Post{
			BlogID:  blog.BlogID,
			Title:   "titleA",
			Content: "contentA",
		})
		dbc.Post.Insert(&Post{
			BlogID:  blog.BlogID,
			Title:   "titleB",
			Content: "contentB",
		})
	}(src[0])
	src[1].Author = "unknown"
	dbc.Blog.Update(src[1])
	dbc.Blog.Delete(src[2])
	if err := dbc.SaveChanges(); err != nil {
		panic(err)
	}

	// Output:
	// blogs = 3
	// (*example.Blog){BlogID:(uuid.UUID)d03bc237-eef4-4b6f-afe1-ea901357d828 Name:(string)testing1 Author:(string)kamichidu Posts:([]*example.Post)[<max>]}
	// - (*example.Post){BlogID:(uuid.UUID)d03bc237-eef4-4b6f-afe1-ea901357d828 PostID:(int)1 Title:(string)titleA Content:(string)contentA Blog:(*example.Blog){<max>}}
	// - (*example.Post){BlogID:(uuid.UUID)d03bc237-eef4-4b6f-afe1-ea901357d828 PostID:(int)2 Title:(string)titleB Content:(string)contentB Blog:(*example.Blog){<max>}}
	// (*example.Blog){BlogID:(uuid.UUID)b95e5d4d-7eb9-4612-882d-224daa4a59ee Name:(string)testing2 Author:(string)unknown Posts:([]*example.Post)<nil>}
	// (*example.Blog){BlogID:(uuid.UUID)065c6554-9aff-4b42-ab3b-141ed5ef5624 Name:(string)testing4 Author:(string)kamichidu Posts:([]*example.Post)<nil>}
	blogs, err := dbc.Blog.Select().
		Include(dbc.Blog.IncludePosts, dbc.Post.IncludeBlog).
		Where(dbc.Blog.Name.Like(`%testing%`)).
		OrderBy(dbc.Blog.Name.Asc()).
		Query()
	if err != nil {
		panic(err)
	}
	fmt.Printf("blogs = %d\n", len(blogs))
	spew.Config.SortKeys = true
	spew.Config.MaxDepth = 1
	for _, blog := range blogs {
		spew.Printf("%#v\n", blog)

		for _, post := range blog.Posts {
			spew.Printf("- %#v\n", post)
		}
	}
}
