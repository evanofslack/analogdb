package weaviate

import (
	"context"
	"fmt"
	"time"

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
}

func NewDB(host string, scheme string) *DB {
	db := &DB{host: host, scheme: scheme, timeout: weaviateClientTimeout}
	db.ctx, db.cancel = context.WithCancel(context.Background())
	return db
}

func (db *DB) Open() error {

	// validate host and scheme are set
	if db.host == "" {
		return fmt.Errorf("DB host must be set")
	}
	if db.scheme == "" {
		return fmt.Errorf("DB scheme must be set")
	}

	cfg := weaviate.Config{
		Host:           db.host,
		Scheme:         db.scheme,
		StartupTimeout: db.timeout,
	}

	var err error
	db.db, err = weaviate.NewClient(cfg)
	if err != nil {
		return err
	}
	return err
}

func (db *DB) Migrate(ctx context.Context) error {
	schema, err := db.getSchema(ctx)
	if err != nil {
		fmt.Println("failed to get weaviate schema")
		return err
	}
	// if no classes, create schemas
	if len(schema.Classes) == 0 {
		if err := db.createSchemas(ctx); err != nil {
			fmt.Println("failed to create weaviate schema")
			return err
		}
	}
	return nil
}

func (db *DB) Close() error {
	db.cancel()
	return nil
}
