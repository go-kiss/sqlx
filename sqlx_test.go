package sqlx

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

var schema = `
CREATE TABLE IF NOT EXISTS users (
  id integer primary key,
  age integer,
  name varchar(30),
  created datetime default CURRENT_TIMESTAMP
)
`

type user struct {
	ID      int
	Name    string
	Age     int
	Created time.Time
}

func (u *user) TableName() string { return "users" }
func (u *user) KeyName() string   { return "id" }

func TestCRUD(t *testing.T) {
	ctx := context.Background()

	db := DB{sqlx.MustOpen("sqlite", ":memory:")}
	db.MustExecContext(ctx, schema)

	now := time.Now()
	u1 := &user{Name: "foo", Age: 18, Created: now}
	result, err := db.Insert(u1)
	if err != nil {
		t.Fatal(err)
	}

	id, _ := result.LastInsertId()

	var u2 user
	err = db.Get(&u2, "select * from users where id = ?", id)
	if err != nil {
		t.Fatal(err)
	}

	if u2.Name != "foo" || u2.Age != 18 || !u2.Created.Equal(now) {
		t.Fatal("invalid user", u2)
	}

	u2.Name = "bar"
	_, err = db.Update(&u2)
	if err != nil {
		t.Fatal(err)
	}

	var u3 user
	err = db.Get(&u3, "select * from users where id = ?", id)
	if err != nil {
		t.Fatal(err)
	}

	if u3.Name != "bar" || u3.Age != 18 || !u3.Created.Equal(now) {
		t.Fatal("invalid user", u3)
	}
	u4 := *u1
	u4.ID = 10
	_, err = db.Insert(&u4)
	if err != nil {
		t.Fatal(err)
	}

	var u5 user
	err = db.Get(&u5, "select * from users where id = ?", u4.ID)
	if err != nil {
		t.Fatal(err)
	}

	if u5.ID != u4.ID {
		t.Fatal("invalid user", u5)
	}
}
