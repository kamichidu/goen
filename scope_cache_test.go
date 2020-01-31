package goen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type Email struct {
	EmailID int64 `goen:""`

	Email string

	User *User `foreign_key:"email_id"`
}

type User struct {
	UserID int64 `goen:"" primary_key:""`

	Name string

	EmailID int64

	Email *Email `foreign_key:"email_id"`
}

func TestScopeCache(t *testing.T) {
	meta := NewMetaSchema()
	meta.Register(new(Email))
	meta.Register(new(User))
	meta.Compute()

	sc := NewScopeCache(meta)
	userKey := &MapRowKey{
		Table: "user",
		Key: map[string]interface{}{
			"user_id": 1,
		},
	}
	userEmailKey := &MapRowKey{
		Table: "user",
		Key: map[string]interface{}{
			"email_id": 2,
		},
	}
	// XXX: one to one relation, currently assume many to one cardinality
	assert.Nil(t, sc.GetObject(CardinalityNone, userKey), "user's user_id key is not cached yet")
	assert.Nil(t, sc.GetObject(CardinalityManyToOne, userEmailKey), "user's email_id key is not cached yet")
	assert.Equal(t, false, sc.HasObject(CardinalityNone, userKey), "user's user_id key is not cached yet")
	assert.Equal(t, false, sc.HasObject(CardinalityManyToOne, userEmailKey), "user's email_id key is not cached yet")
	assert.NotPanics(t, func() {
		sc.RemoveObject(&User{
			UserID:  1,
			EmailID: 2,
		})
	})

	user := &User{
		UserID:  1,
		Name:    "testing",
		EmailID: 2,
	}
	sc.AddObject(user)

	assert.Exactly(t, user, sc.GetObject(CardinalityNone, userKey), "GetObject returns cached entity by user's user_id key")
	assert.Exactly(t, user, sc.GetObject(CardinalityManyToOne, userEmailKey), "GetObject returns cached entity by user's email_id key")
	assert.Equal(t, true, sc.HasObject(CardinalityNone, userKey), "HasObject returns true if cached by user's user_id key")
	assert.Equal(t, true, sc.HasObject(CardinalityManyToOne, userEmailKey), "HasObject returns true if cached by user's email_id key")
	assert.NotPanics(t, func() {
		sc.RemoveObject(&User{
			UserID:  1,
			EmailID: 2,
		})
	})
	assert.Nil(t, sc.GetObject(CardinalityNone, userKey), "user's user_id key was deleted")
	assert.Nil(t, sc.GetObject(CardinalityManyToOne, userEmailKey), "user's email_id key was deleted")
	assert.Equal(t, false, sc.HasObject(CardinalityNone, userKey), "user's user_id key was deleted")
	assert.Equal(t, false, sc.HasObject(CardinalityManyToOne, userEmailKey), "user's email_id key was deleted")
}
