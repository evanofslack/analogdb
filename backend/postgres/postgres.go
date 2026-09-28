package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/evanofslack/analogdb/logger"
	"github.com/evanofslack/analogdb/metrics"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
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
	pool             Pool
}

// Pool holds connection pool limits. Zero values keep the database/sql defaults.
type Pool struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// SetPool sets the connection pool limits applied on Open.
func (db *DB) SetPool(pool Pool) {
	db.pool = pool
}

func NewDB(dsn string, logger *logger.Logger, migrationEnabled bool, migrationPath string, tracingEnabled bool) *DB {
	host, name := dsnHostAndName(dsn)
	logger.Debug("Initializing db instance", "migration_path", migrationPath, "migration_enabled", migrationEnabled, "host", host, "db_name", name)
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
		if err != nil {
			return fmt.Errorf("register db tracing driver: %w", err)
		}
		db.logger.Info("Instrumented db with tracing")
	}

	if db.db, err = sql.Open(driver, db.dsn); err != nil {
		err = fmt.Errorf("open connection to db: %w", err)
		return err
	}
	db.applyPool()

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

func (db *DB) applyPool() {
	if db.pool.MaxOpenConns > 0 {
		db.db.SetMaxOpenConns(db.pool.MaxOpenConns)
	}
	if db.pool.MaxIdleConns > 0 {
		db.db.SetMaxIdleConns(db.pool.MaxIdleConns)
	}
	if db.pool.ConnMaxLifetime > 0 {
		db.db.SetConnMaxLifetime(db.pool.ConnMaxLifetime)
	}
	if db.pool.ConnMaxIdleTime > 0 {
		db.db.SetConnMaxIdleTime(db.pool.ConnMaxIdleTime)
	}
	db.logger.Debug("Set db pool", "max_open_conns", db.pool.MaxOpenConns, "max_idle_conns", db.pool.MaxIdleConns, "conn_max_lifetime", db.pool.ConnMaxLifetime, "conn_max_idle_time", db.pool.ConnMaxIdleTime)
}

// RegisterMetrics exposes connection pool stats as analogdb_go_sql_* series.
func (db *DB) RegisterMetrics(registerer prometheus.Registerer) error {
	collector := collectors.NewDBStatsCollector(db.db, metrics.AnalogdbNamespace)
	return prometheus.WrapRegistererWithPrefix(metrics.AnalogdbNamespace+"_", registerer).Register(collector)
}

func (db *DB) Close() error {
	db.logger.Debug("Starting to close db connection")

	db.cancel()

	if db.db != nil {
		if err := db.db.Close(); err != nil {
			return err
		}
	}

	db.logger.Info("Closed db connection")
	return nil
}

// dsnHostAndName returns the host and database name from a dsn
func dsnHostAndName(dsn string) (string, string) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", ""
	}
	return u.Host, strings.TrimPrefix(u.Path, "/")
}
