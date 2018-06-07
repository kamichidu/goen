# goen

goen is a typesafe GOlang ENtity interface for relational databases.

It provides a way of programmatically to interact with relational databases.
It aims to implement in a go way and not to provide fully ORM features.

## Installation

```
go get -u github.com/kamichidu/goen/...
```

goen provides a binary tool used by go generate, please be sure that `$GOPATH/bin` is on your `$PATH` .

## Usage

Write your first entity is.

```
package entity

type User struct {
    UserID int `goen:"" pk:""`
    Name, Email, PasswordHash string
}
```

Then put the following on any file of that package:

```
//go:generate goen -o goen.go
```

Now, all you have to do is run `go generate ./...` and a `goen.go` file will be generated.

## Define entities

An entity is just a go struct have a struct tag `goen:""` .
All fields of this struct will be columns in the database table.
An entity also needs to have one primary key.
The primary key is defined using the `pk:""` struct tag on the primary key fields.

Let's review the rules and conventions for entity fields:

- All the fields with basic types or types that implement sql.Scanner and driver.Valuer will be considered a column in the table of their matching type.
- By default, the name of a table/view will be the name of the struct converted to lower snake case (e.g.`User` => `user`, `UserFriend` => `user_friend`)
- By default, the name of a column will be the name of the struct field converted to lower snake case (e.g. `UserName` => `user_name`, `UserID` => `user_id`). You can override it with the struct tag `column:"custom_name"`.

## Struct tags

| Tag | Description |
| --- | --- |
| `goen:""` | Indicates this struct as an entity. goen finds structs that have this struct tag. |
| `table:"table_name"` | Specifies a table name. |
| `view:"view_name"` | Specifies a view name for readonly entity. |
| `primary_key:""` | |
| `primary_key:"column_name"` | |
| `primary_key:"column_name,omitempty"` | |
| `primary_key:",omitempty"` | |
| `column:"column_name"` | |
| `column:"column_name,omitempty"` | |
| `column:",omitempty"` | |
| `foreign_key:"column_name"` | |
| `foreign_key:"column_name1,column_name2:reference_column_name"` | |
| `ignore:""` | |

## TODO until alpha release

- [x] write license header
- [ ] escape table and column name
- [ ] <s>tracking changes for entity</s>
- [ ] <s>detaching</s>
- [ ] <s>auto detect insert/update</s>
    - <s>needs to insert/update related entities</s>
- [x] auto loading relations
- [x] eager loading
- [x] transaction support
- [x] entity caching
- [x] able to output a generated file to an another directory
- [ ] embedded struct support
