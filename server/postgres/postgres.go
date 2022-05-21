package postgres

import (
	"context"
	"database/sql"
)

type DB struct {
	db     *sql.DB
	ctx    context.Context
	cancel func()
	prod   bool
}

func NewDB() *DB {
	db := &DB{}
	db.ctx, db.cancel = context.WithCancel(context.Background())
}
