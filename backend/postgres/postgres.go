package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/evanofslack/analogdb/logger"
	_ "github.com/lib/pq"
	"go.nhat.io/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

type DB struct {
	db             *sql.DB
	dsn            string
	ctx            context.Context
	cancel         func()
	logger         *logger.Logger
	tracingEnabled bool
}

func NewDB(dsn string, logger *logger.Logger, tracingEnabled bool) *DB {
	logger.Debug().Msg("Initializing DB instance")

	ctx, cancel := context.WithCancel(context.Background())

	db := &DB{
		dsn:            dsn,
		ctx:            ctx,
		cancel:         cancel,
		logger:         logger,
		tracingEnabled: tracingEnabled,
	}

	db.logger.Info().Msg("Initialized DB instance")

	return db
}

func (db *DB) Open() error {
	db.logger.Debug().Msg("Opening DB instance")

	if db.dsn == "" {
		return fmt.Errorf("data source name name must be set for DB")
	}

	var err error
	driver := "postgres"

	if db.tracingEnabled {
		driver, err = otelsql.Register("postgres",
			otelsql.TraceQueryWithoutArgs(),
			otelsql.TraceRowsClose(),
			otelsql.TraceRowsAffected(),
			otelsql.WithDatabaseName("analogdb"),
			otelsql.WithSystem(semconv.DBSystemPostgreSQL),
		)
		db.logger.Info().Msg("Instrumented DB with tracing")
	}

	if db.db, err = sql.Open(driver, db.dsn); err != nil {
		err = fmt.Errorf("open connection to DB: %w", err)
		return err
	}

	db.logger.Info().Msg("Opened new DB instance")

	return db.db.PingContext(db.ctx)
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
