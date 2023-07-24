package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/evanofslack/analogdb/logger"
	_ "github.com/lib/pq"
)

type DB struct {
	db     *sql.DB
	dsn    string
	ctx    context.Context
	cancel func()
	logger *logger.Logger
}

func NewDB(dsn string, logger *logger.Logger) *DB {
	db := &DB{dsn: dsn, logger: logger}
	db.ctx, db.cancel = context.WithCancel(context.Background())
	db.logger.Info().Msg("Initialized DB instance")
	return db
}

func (db *DB) Open() error {
	if db.dsn == "" {
		return fmt.Errorf("DB data source name must be set")
	}
	var err error
	if db.db, err = sql.Open("postgres", db.dsn); err != nil {
		err = fmt.Errorf("Failed to open connection to DB: %w", err)
		return err
	}
	go db.monitor()

	db.logger.Info().Msg("Opened new DB connection")
	return db.db.Ping()
}

func (db *DB) Close() error {

	db.logger.Debug().Msg("Starting to close DB connection")

	db.cancel()

	if db.db != nil {
		db.db.Close()
	}

	db.logger.Info().Msg("Closed DB connection")
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
