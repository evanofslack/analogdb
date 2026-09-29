package clickhouse

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/logger"
)

const (
	queryTimeout       = 5 * time.Second
	maxExecutionTime   = 5
	defaultTable       = "httprequests"
	dialTimeout        = 5 * time.Second
	maxOpenConns       = 4
	maxIdleConns       = 2
	connMaxLifetime    = time.Hour
	clientProductName  = "analogdb"
	clientProductBuild = "admin"
)

var tableName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ensure interface is implemented
var _ analogdb.AnalyticsService = (*DB)(nil)

type DB struct {
	conn     driver.Conn
	addr     string
	database string
	username string
	password string
	table    string
	logger   *logger.Logger
}

func NewDB(host string, port int, database, username, password, table string, logger *logger.Logger) (*DB, error) {
	if host == "" {
		return nil, fmt.Errorf("clickhouse host must be set")
	}
	if table == "" {
		table = defaultTable
	}
	if !tableName.MatchString(table) {
		return nil, fmt.Errorf("invalid clickhouse table name %q", table)
	}
	db := &DB{
		addr:     fmt.Sprintf("%s:%d", host, port),
		database: database,
		username: username,
		password: password,
		table:    table,
		logger:   logger,
	}
	db.logger.Info("Initialized clickhouse instance", "addr", db.addr, "database", database, "table", table)
	return db, nil
}

func (db *DB) Open() error {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{db.addr},
		Auth: clickhouse.Auth{
			Database: db.database,
			Username: db.username,
			Password: db.password,
		},
		ClientInfo: clickhouse.ClientInfo{
			Products: []struct {
				Name    string
				Version string
			}{
				{Name: clientProductName, Version: clientProductBuild},
			},
		},
		Settings: clickhouse.Settings{
			"max_execution_time": maxExecutionTime,
		},
		DialTimeout:     dialTimeout,
		MaxOpenConns:    maxOpenConns,
		MaxIdleConns:    maxIdleConns,
		ConnMaxLifetime: connMaxLifetime,
	})
	if err != nil {
		return fmt.Errorf("open clickhouse: %w", err)
	}
	db.conn = conn
	db.logger.Info("Opened clickhouse connection")
	return nil
}

func (db *DB) Readyz(ctx context.Context) error {
	return db.conn.Ping(ctx)
}

func (db *DB) Close() error {
	if db.conn == nil {
		return nil
	}
	return db.conn.Close()
}

// query runs with the query timeout, call done to close the rows
func (db *DB) query(ctx context.Context, query string, args ...any) (driver.Rows, func(), error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	rows, err := db.conn.Query(ctx, query, args...)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	done := func() {
		if err := rows.Close(); err != nil {
			db.logger.Warn("Fail close clickhouse rows", "error", err)
		}
		cancel()
	}
	return rows, done, nil
}
