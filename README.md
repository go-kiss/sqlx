# sqlx

Extended sqlx supporting Insert/Save.

In order to support insert and update, we need a way to get the table name.

One simplest way is by function parameter, so we may use the following signature

```go
func Insert(table string, arg interface{}) (sql.Result, error)
```

So far so good. However, if we want to support update, we need a way to indicate the filter condition(maybe primary key).

So we may use the following signature

```go
func Insert(table, key string, arg interface{}) (sql.Result, error)
```

According to my experiment, even Insert needs to know which key used as primary key. So I introduced the Modeler interface

```go
type Modeler interface {
  TableName() string
  KeyName() string
}
```

And the above functions become like this

```go
func Insert(arg Modeler) (sql.Result, error)
func Update(arg Modeler) (sql.Result, error)
```

A full example is like this

```go
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

u1 := &user{Name: "foo", Age: 18, Created: now}
// INSERT INTO users(name,age,created) VALUES (?,?,?)
result, err := db.Insert(u1)

id, _ := result.LastInsertId()

u1.Name = "bar"
u1.ID = int(id)

// UPDATE users SET name=?,age=?,created=? WHERE id=?
_, err = db.Update(u1)
```
