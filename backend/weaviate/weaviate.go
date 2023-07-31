package weaviate

import (
	"context"
	"fmt"
	"time"

	"github.com/evanofslack/analogdb/logger"
	"github.com/evanofslack/analogdb/tracer"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
	tracer  *tracer.Tracer
}

func NewDB(host string, scheme string, logger *logger.Logger, tracer *tracer.Tracer) *DB {
	db := &DB{
		host:    host,
		scheme:  scheme,
		timeout: weaviateClientTimeout,
		logger:  logger,
		tracer:  tracer,
	}
	db.ctx, db.cancel = context.WithCancel(context.Background())
	db.logger.Info().Msg("Initialized vector DB instance")
	return db
}

func (db *DB) Open() error {

	db.logger.Debug().Msg("Starting vector DB open")

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
		db.logger.Error().Err(err).Msg("Failed to open vector DB")
		return err
	}

	db.logger.Info().Msg("Opened new vector DB connection")
	return err
}

func (db *DB) Migrate(ctx context.Context) error {

	db.logger.Debug().Msg("Starting vector DB migration")

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
	db.logger.Info().Msg("Completed vector DB migration")
	return nil
}

func (db *DB) Close() error {
	db.logger.Debug().Msg("Starting vector DB close")
	db.cancel()
	db.logger.Info().Msg("Closed vector DB connection")
	return nil
}

// start a trace targeting the weaviate server
func (db *DB) startTrace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {

	dbSystem := attribute.String("db.system", "weaviate")
	dbName := attribute.String("db.name", "pictures")
	serverAddress := attribute.String("server.address", db.host)
	spanKind := trace.WithSpanKind(trace.SpanKindClient)

	opts = append(opts, dbSystem, dbName, serverAddress, spanKind)

	return db.tracer.Tracer.Start(ctx, spanName, opts)

}
