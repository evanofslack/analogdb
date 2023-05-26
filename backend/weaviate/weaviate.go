package weaviate

import (
	"context"
	"fmt"
	"time"

	"github.com/evanofslack/analogdb/logger"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
)

const weaviateClientTimeout = 30 * time.Second

type DB struct {
	db      *weaviate.Client
	host    string
	scheme  string
	timeout time.Duration
	ctx     context.Context
	cancel  func()
	logger  *logger.Logger
}

func NewDB(host string, scheme string, logger *logger.Logger) *DB {
	db := &DB{host: host, scheme: scheme, timeout: weaviateClientTimeout, logger: logger}
	db.ctx, db.cancel = context.WithCancel(context.Background())
	db.logger.Logger.Info().Msg("Initialized vector DB instance")
	return db
}

func (db *DB) Open() error {

	// validate host and scheme are set
	if db.host == "" {
		return fmt.Errorf("Vector DB host must be set")
	}
	if db.scheme == "" {
		return fmt.Errorf("Vector DB scheme must be set")
	}

	cfg := weaviate.Config{
		Host:           db.host,
		Scheme:         db.scheme,
		StartupTimeout: db.timeout,
	}

	var err error
	db.db, err = weaviate.NewClient(cfg)
	if err != nil {
		err = fmt.Errorf("Failed to create new vector DB client: %w", err)
		return err
	}

	db.logger.Logger.Info().Msg("Opened new vector DB connection")
	return err
}

func (db *DB) Migrate(ctx context.Context) error {
	schema, err := db.getSchema(ctx)
	if err != nil {
		err = fmt.Errorf("Failed to get weaviate schema: %w", err)
		return err
	}
	// if no classes, create schemas
	if len(schema.Classes) == 0 {
		if err := db.createSchemas(ctx); err != nil {
			err = fmt.Errorf("Failed to get create schema: %w", err)
			return err
		}
	}
	db.logger.Logger.Info().Msg("Completed vector DB migration")
	return nil
}

func (db *DB) Close() error {
	db.cancel()
	db.logger.Logger.Info().Msg("Closed vector DB connection")
	return nil
}
