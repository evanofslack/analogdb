package clickhouse

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	v1 "github.com/evanofslack/analogdb-consumer/internal/gen/proto/analytics/v1"
)

type DB interface {
	Insert(ctx context.Context, events []*v1.Event) error
	Health(ctx context.Context) error
	Close() error
}

type Client struct {
	logger           *slog.Logger
	conn             clickhouse.Conn
	addr             string
	database         string
	username         string
	password         string
	table            string
	migrationEnabled bool
	migrationPath    string
	appName          string
	appVersion       string
}

func New(host string, port int, database, username, password, table string, logger *slog.Logger, appName, appVersion string, migrationEnabled bool, migrationPath string) (*Client, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	logger.Debug("Initializing clickhouse instance", "migration_path", migrationPath, "migration_enabled", migrationEnabled, "addr", addr, "table", table)

	client := &Client{
		logger:           logger,
		addr:             addr,
		database:         database,
		username:         username,
		password:         password,
		table:            table,
		migrationEnabled: migrationEnabled,
		migrationPath:    migrationPath,
		appName:          appName,
		appVersion:       appVersion,
	}
	if err := client.Health(context.Background()); err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	logger.Info("Initialized clickhouse instance", "migration_path", migrationPath, "migration_enabled", migrationEnabled, "addr", addr, "table", table)
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
	return nil
}

func (c *Client) Insert(ctx context.Context, events []*v1.Event) error {
	if len(events) == 0 {
		return nil
	}

	batch, err := c.conn.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s", c.table))
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
		return fmt.Errorf("send batch: %w", err)
	}

	c.logger.Debug("inserted events", "count", len(events))
	return nil
}

func (c *Client) Health(ctx context.Context) error {
	return c.conn.Ping(ctx)
}

func (c *Client) Close() error {
	return c.conn.Close()
}
