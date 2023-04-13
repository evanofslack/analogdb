package weaviate

import (
	"context"
	"fmt"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
)

type DB struct {
	db     *weaviate.Client
	host   string
	scheme string
	ctx    context.Context
	cancel func()
}

func NewDB(host string, scheme string) *DB {
	db := &DB{host: host, scheme: scheme}
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
		Host:   db.host,
		Scheme: db.scheme,
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
		return err
	}
	// if no classes, create schemas
	if len(schema.Classes) == 0 {
		if err := db.createSchemas(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) Close() error {
	db.cancel()
	return nil
}
