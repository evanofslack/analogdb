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
	db               *sql.DB
	dsn              string
	ctx              context.Context
	cancel           func()
	logger           *logger.Logger
	migrationEnabled bool
	migrationPath    string
	tracingEnabled   bool
}

func NewDB(dsn string, logger *logger.Logger, migrationEnabled bool, migrationPath string, tracingEnabled bool) *DB {
	logger.Debug("Initializing db instance", "migration_path", migrationPath, "migration_enabled", migrationEnabled, "dsn", dsn)
	ctx, cancel := context.WithCancel(context.Background())

	db := &DB{
		dsn:              dsn,
		ctx:              ctx,
		cancel:           cancel,
		logger:           logger,
		migrationEnabled: migrationEnabled,
		migrationPath:    migrationPath,
		tracingEnabled:   tracingEnabled,
	}
	db.logger.Info("Initialized db instance")
	return db
}

func (db *DB) Open() error {
	db.logger.Debug("Opening db instance")
	defer db.logger.Info("Opened db instance", "migration_enabled", db.migrationEnabled)

	if db.dsn == "" {
		return fmt.Errorf("data source name name must be set for db")
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
		db.logger.Info("Instrumented db with tracing")
	}

	if db.db, err = sql.Open(driver, db.dsn); err != nil {
		err = fmt.Errorf("open connection to db: %w", err)
		return err
	}

	if db.migrationEnabled {
		// If migration path provided, use it
		if db.migrationPath != "" {
			if err := db.migrateFromPath(db.migrationPath); err != nil {
				return err
			}
		} else {
			// Otherwise use default embedded migrations
			if err := db.migrate(); err != nil {
				return err
			}
		}
	}
	return db.db.PingContext(db.ctx)
}

func (db *DB) Close() error {
	db.logger.Debug("Starting to close db connection")

	db.cancel()

	if db.db != nil {
		db.db.Close()
	}

	db.logger.Info("Closed db connection")
	return nil
}
