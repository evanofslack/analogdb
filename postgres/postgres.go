package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	_ "github.com/lib/pq"
)

type DB struct {
	db     *sql.DB
	dsn    string
	ctx    context.Context
	cancel func()
}

func NewDB(dsn string) *DB {
	db := &DB{dsn: dsn}
	db.ctx, db.cancel = context.WithCancel(context.Background())
	return db
}

func (db *DB) Open() error {
	if db.dsn == "" {
		return fmt.Errorf("Data source name must be set")
	}
	var err error
	if db.db, err = sql.Open("postgres", db.dsn); err != nil {
		return err
	}
	go db.monitor()

	return db.db.Ping()
}

func (db *DB) Close() error {
	db.cancel()

	if db.db != nil {
		db.db.Close()
	}
	return nil
}

func (db *DB) monitor() {
	// add prometheous metrics
}

// NullString is an alias for sql.NullString data type
type NullString sql.NullString

// Scan implements the Scanner interface for NullString
func (ns *NullString) Scan(value interface{}) error {
	var s sql.NullString
	if err := s.Scan(value); err != nil {
		return err
	}

	// if nil then make Valid false
	if reflect.TypeOf(value) == nil {
		*ns = NullString{s.String, false}
	} else {
		*ns = NullString{s.String, true}
	}

	return nil
}
