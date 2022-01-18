package sqlx

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

// Modeler provides information of table.
// Objects support Insert/Update should implement this interface.
type Modeler interface {
	// TableName return table name.
	TableName() string
	// TableName return primary key name.
	KeyName() string
}

// DB extends the original sqlx.DB
type DB struct {
	*sqlx.DB
}

// Tx extends the original sqlx.Tx
type Tx struct {
	*sqlx.Tx
}

// mapExecer unifies DB and TX
type mapExecer interface {
	DriverName() string
	GetMapper() *reflectx.Mapper
	Rebind(string) string
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// MustBegin return our extended *Tx
func (db *DB) MustBegin() *Tx {
	tx := db.DB.MustBegin()
	return &Tx{tx}
}

// Beginx return our extended *Tx
func (db *DB) Beginx() (*Tx, error) {
	tx, err := db.DB.Beginx()
	if err != nil {
		return nil, err
	}
	return &Tx{tx}, nil
}

// BeginTxx return our extended *Tx
func (db *DB) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.DB.BeginTxx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Tx{tx}, nil
}

// InsertContext generates and executes insert query.
func (db *DB) InsertContext(ctx context.Context, m Modeler) (sql.Result, error) {
	return insert(ctx, db, m)
}

// InsertContext generates and executes insert query without context.
func (db *DB) Insert(m Modeler) (sql.Result, error) {
	return db.InsertContext(context.Background(), m)
}

// UpdateContext generates and executes update query.
func (db *DB) UpdateContext(ctx context.Context, m Modeler) (sql.Result, error) {
	return update(ctx, db, m)
}

// UpdateContext generates and executes update query without context.
func (db *DB) Update(m Modeler) (sql.Result, error) {
	return db.UpdateContext(context.Background(), m)
}

// InsertContext generates and executes insert query.
func (tx *Tx) InsertContext(ctx context.Context, m Modeler) (sql.Result, error) {
	return insert(ctx, tx, m)
}

// InsertContext generates and executes insert query without context.
func (tx *Tx) Insert(m Modeler) (sql.Result, error) {
	return tx.InsertContext(context.Background(), m)
}

// UpdateContext generates and executes update query.
func (tx *Tx) UpdateContext(ctx context.Context, m Modeler) (sql.Result, error) {
	return update(ctx, tx, m)
}

// UpdateContext generates and executes update query without context.
func (tx *Tx) Update(m Modeler) (sql.Result, error) {
	return tx.UpdateContext(context.Background(), m)
}

// GetMapper return the Mapper object.
func (db *DB) GetMapper() *reflectx.Mapper {
	return db.Mapper
}

// GetMapper return the Mapper object.
func (tx *Tx) GetMapper() *reflectx.Mapper {
	return tx.Mapper
}

func insert(ctx context.Context, db mapExecer, m Modeler) (sql.Result, error) {
	names, args, err := bindModeler(m, db.GetMapper())
	if err != nil {
		return nil, err
	}

	marks := ""
	k := -1
	for i := 0; i < len(names); i++ {
		if names[i] == m.KeyName() {
			v := reflect.ValueOf(args[i])
			if v.IsZero() {
				k = i
				args = append(args[:i], args[i+1:]...)
				continue
			}
		}
		marks += "?,"
	}
	if k >= 0 {
		names = append(names[:k], names[k+1:]...)
	}
	marks = marks[:len(marks)-1]
	query := "INSERT INTO " + m.TableName() + "(" + strings.Join(names, ",") + ") VALUES (" + marks + ")"
	query = db.Rebind(query)
	return db.ExecContext(ctx, query, args...)
}

func update(ctx context.Context, db mapExecer, m Modeler) (sql.Result, error) {
	names, args, err := bindModeler(m, db.GetMapper())
	if err != nil {
		return nil, err
	}

	query := "UPDATE " + m.TableName() + " set "
	var id interface{}
	for i := 0; i < len(names); i++ {
		name := names[i]
		if name == m.KeyName() {
			id = args[i]
			args = append(args[:i], args[i+1:]...)
			continue
		}
		query += name + "=?,"
	}
	query = query[:len(query)-1] + " WHERE " + m.KeyName() + " = ?"
	query = db.Rebind(query)
	args = append(args, id)
	return db.ExecContext(ctx, query, args...)
}

func bindModeler(arg interface{}, m *reflectx.Mapper) ([]string, []interface{}, error) {
	t := reflect.TypeOf(arg)
	names := []string{}
	for k := range m.TypeMap(t).Names {
		names = append(names, k)
	}
	sort.Stable(sort.StringSlice(names))
	args, err := bindArgs(names, arg, m)
	if err != nil {
		return nil, nil, err
	}

	return names, args, nil
}

func bindArgs(names []string, arg interface{}, m *reflectx.Mapper) ([]interface{}, error) {
	arglist := make([]interface{}, 0, len(names))

	// grab the indirected value of arg
	v := reflect.ValueOf(arg)
	for v = reflect.ValueOf(arg); v.Kind() == reflect.Ptr; {
		v = v.Elem()
	}

	err := m.TraversalsByNameFunc(v.Type(), names, func(i int, t []int) error {
		if len(t) == 0 {
			return fmt.Errorf("could not find name %s in %#v", names[i], arg)
		}

		val := reflectx.FieldByIndexesReadOnly(v, t)
		arglist = append(arglist, val.Interface())

		return nil
	})

	return arglist, err
}
