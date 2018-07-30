[![wercker status](https://app.wercker.com/status/12a1429eafda5aafa0d10f4946551e37/s/master "wercker status")](https://app.wercker.com/project/byKey/12a1429eafda5aafa0d10f4946551e37)
[![Coverage Status](https://coveralls.io/repos/github/kamichidu/goen/badge.svg)](https://coveralls.io/github/kamichidu/goen)
[![godoc](https://godoc.org/github.com/kamichidu/goen?status.svg)](https://godoc.org/github.com/kamichidu/goen)

# goen

goen is a typesafe GOlang ENtity interface for relational databases.

It provides a way of programmatically to interact with relational databases.
It aims to implement in a go way and not to provide fully ORM features.

goen has following concepts:

- No any fields or methods introducing into user's struct
- Work with plain go's `database/sql`
- Based on RTTI
- Support bulk operation
- Go generate is used only for utilities, it's not a core logic

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
    UserID int `goen:"" primary_key:""`
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
The primary key is defined using the `primary_key:""` struct tag on the primary key fields.

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
| `primary_key:""` | Indicates this field is a part of primary key |
| `primary_key:"column_name"` | Indicates this field is a part of primary key and specifies a column name |
| `primary_key:"column_name,omitempty"` | Indicates this field is a part of primary key, specifies a column name and this field is omitting if empty |
| `primary_key:",omitempty"` | Indicates this field is a part of primary key, and this field is omitting if empty |
| `column:"column_name"` | Specifies a column name |
| `column:"column_name,omitempty"` | Specifies a column name and this field is omitting if empty |
| `column:",omitempty"` | Specifies this field is omitting if empty |
| `foreign_key:"column_name"` | Indicates this field is referencing another entity, and specifies keys |
| `foreign_key:"column_name1,column_name2:reference_column_name"` | Indicates this field is referencing another entity, and specifies key pairs |
| `ignore:""` | Specifies this columns is to be ignored |
