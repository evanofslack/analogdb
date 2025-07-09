package clickhouse

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	v1 "github.com/evanofslack/analogdb-consumer/internal/gen/proto/analytics/v1"
	"github.com/evanofslack/analogdb-consumer/internal/metrics"
)

type DB interface {
	Insert(ctx context.Context, events []*v1.Event) error
	Health(ctx context.Context) error
	Close() error
}

type Client struct {
	logger           *slog.Logger
	metrics          *metrics.Metrics
	conn             clickhouse.Conn
	addr             string
	dsn              string
	database         string
	username         string
	password         string
	table            string
	migrationEnabled bool
	migrationPath    string
	appName          string
	appVersion       string
}

func New(logger *slog.Logger, metrics *metrics.Metrics, host string, port int, database, username, password, table string, appName, appVersion string, migrationEnabled bool, migrationPath string) (*Client, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	dsn := fmt.Sprintf("clickhouse://%s:%s@%s:%d/%s", username, password, host, port, database)
	logger = logger.With("addr", addr, "dsn", dsn, "table", table)
	logger.Debug("Start init clickhouse instance", "migration_path", migrationPath, "migration_enabled", migrationEnabled)

	client := &Client{
		logger:           logger,
		metrics:          metrics,
		addr:             addr,
		dsn:              dsn,
		database:         database,
		username:         username,
		password:         password,
		table:            table,
		migrationEnabled: migrationEnabled,
		migrationPath:    migrationPath,
		appName:          appName,
		appVersion:       appVersion,
	}
	logger.Info("Finish init clickhouse instance", "migration_path", migrationPath, "migration_enabled", migrationEnabled)
	return client, nil
}

func (c *Client) Open() error {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{c.addr},
		Auth: clickhouse.Auth{
			Database: c.database,
			Username: c.username,
			Password: c.password,
		},
		ClientInfo: clickhouse.ClientInfo{
			Products: []struct {
				Name    string
				Version string
			}{
				{Name: c.appName, Version: c.appVersion},
			},
		},
		Debugf: func(format string, v ...interface{}) {
			c.logger.Debug(fmt.Sprintf(format, v...))
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout:      time.Second * 30,
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Hour,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
	})
	if err != nil {
		return fmt.Errorf("connect to clickhouse: %w", err)
	}
	c.conn = conn

	if c.migrationEnabled {
		// If migration path provided, use it
		if c.migrationPath != "" {
			if err := c.migrateFromPath(c.migrationPath); err != nil {
				return err
			}
		} else {
			// Otherwise use default embedded migrations
			if err := c.migrate(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Client) Insert(ctx context.Context, events []*v1.Event) error {
	if len(events) == 0 {
		return nil
	}

	start := time.Now()
	defer c.metrics.ObserveClickHouseInsertDuration(c.table, time.Since(start))
	c.logger.Debug("Start insert events", "count", len(events))

	// Explicitly specify the columns you're inserting
	insert := fmt.Sprintf(`INSERT INTO %s (
		request_id, remote_ip, url, path, protocol, scheme, method, 
		user_agent, response_code, hostname, authorized, start_time, 
		end_time, request_time_ms, bytes_in, bytes_out
	)`, c.table)

	batch, err := c.conn.PrepareBatch(ctx, insert)
	if err != nil {
		return fmt.Errorf("prepare batch: %w", err)
	}

	for _, event := range events {
		err := batch.Append(
			event.RequestId,
			event.RemoteIp,
			event.Url,
			event.Path,
			event.Protocol,
			event.Scheme,
			event.Method,
			event.UserAgent,
			event.ResponseCode,
			event.Hostname,
			event.Authorized,
			event.StartTime,
			event.EndTime,
			event.RequestTimeMs,
			event.BytesIn,
			event.BytesOut,
		)
		if err != nil {
			return fmt.Errorf("append event: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		c.metrics.IncrementClickHouseInserts(len(events), c.table, err)
		return fmt.Errorf("insert batch of events: %w", err)
	}

	c.metrics.IncrementClickHouseInserts(len(events), c.table, nil)
	c.logger.Debug("Finish insert events", "count", len(events))
	return nil
}

func (c *Client) HealthCheck(ctx context.Context) error {
	if c.conn == nil {
		return fmt.Errorf("db connection not open")
	}
	return c.conn.Ping(ctx)
}

func (c *Client) Close() error {
	if c.conn == nil {
		return fmt.Errorf("db connection not open")
	}
	return c.conn.Close()
}
