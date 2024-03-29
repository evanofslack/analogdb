package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/config"
	"github.com/evanofslack/analogdb/logger"
	"github.com/evanofslack/analogdb/metrics"
	"github.com/evanofslack/analogdb/postgres"
	"github.com/evanofslack/analogdb/redis"
	"github.com/evanofslack/analogdb/server"
	"github.com/evanofslack/analogdb/tracer"
	"github.com/evanofslack/analogdb/weaviate"
)

const defaultConfigPath = "config.yml"

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	var cfgPath string
	flag.StringVar(&cfgPath, "config", defaultConfigPath, "path to config.yml")
	flag.Parse()

	// generate the config
	cfg, err := config.New(cfgPath)
	if err != nil {
		err = fmt.Errorf("Failed to parse app config: %w", err)
		fatal(nil, err)
	}

	// create logger instance
	logger, err := logger.New(cfg.Log.Level, cfg.App.Env, cfg.App.Name)
	if err != nil {
		err = fmt.Errorf("Failed to create logger: %w", err)
		fatal(nil, err)
	}
	logger.Info().Str("version", cfg.App.Version).Str("env", cfg.App.Env).Str("loglevel", cfg.Log.Level).Msg("Initializing application")

	// add stack trace
	logger = logger.WithStackTrace()

	// add slack webhook to logger to notify on error
	if webhookURL := cfg.Log.WebhookURL; webhookURL != "" && cfg.App.Env != "debug" {
		logger = logger.WithSlackNotifier(webhookURL)
	}

	// add otel tracing to logger if enabled
	if cfg.Tracing.Enabled {
		logger = logger.WithTracer(cfg.App.Name)
	}

	// initialize otlp tracing
	tracingLogger := logger.WithSubsystem("tracer")
	tracer, err := tracer.New(tracingLogger, cfg)
	if err != nil {
		err = fmt.Errorf("Failed to initialize otlp tracing: %w", err)
		fatal(logger, err)
	}

	if cfg.Tracing.Enabled {
		if err := tracer.StartExporter(); err != nil {
			err = fmt.Errorf("Failed to start otel exporter: %w", err)
			fatal(logger, err)
		}
	}

	// initialize prometheus metrics
	metricsLogger := logger.WithSubsystem("metrics")
	metrics, err := metrics.New(metricsLogger)
	if err != nil {
		err = fmt.Errorf("Failed to initialize prometheus metrics: %w", err)
		fatal(logger, err)
	}

	if cfg.Metrics.Enabled {
		metrics.Serve(cfg.Metrics.Port)
	}

	// open connection to postgres
	dbLogger := logger.WithSubsystem("database")
	db := postgres.NewDB(cfg.DB.URL, dbLogger, cfg.Tracing.Enabled)
	if err := db.Open(); err != nil {
		err = fmt.Errorf("Failed to startup database: %w", err)
		fatal(logger, err)
	}

	// open connection to weaviate
	dbVecLogger := logger.WithSubsystem("vector-database")
	dbVec := weaviate.NewDB(cfg.VectorDB.Host, cfg.VectorDB.Scheme, dbVecLogger, tracer)
	if err := dbVec.Open(); err != nil {
		err = fmt.Errorf("Failed to startup vector database: %w", err)
		fatal(logger, err)
	}
	// run weaviate migrations if needed
	if err := dbVec.Migrate(ctx); err != nil {
		err = fmt.Errorf("Failed to migrate vector database: %w", err)
		fatal(logger, err)
	}

	// open connection to redis if cache enabled
	var rdb *redis.RDB
	if cfg.App.CacheEnabled {
		redisLogger := logger.WithSubsystem("redis")
		rdb, err = redis.NewRDB(cfg.Redis.URL, redisLogger, metrics, cfg.Tracing.Enabled)
		if err != nil {
			err = fmt.Errorf("Failed to startup redis: %w", err)
			fatal(logger, err)
		}
		if err := rdb.Open(); err != nil {
			err = fmt.Errorf("Failed to connect to redis: %w", err)
			fatal(logger, err)
		}
	}

	// initialize http server
	httpLogger := logger.WithSubsystem("http")
	server := server.New(cfg.HTTP.Port, httpLogger, metrics, cfg)

	// need to clean up this dependency injection
	var postService analogdb.PostService
	var authorService analogdb.AuthorService
	var readyService analogdb.ReadyService
	var scrapeService analogdb.ScrapeService
	var keywordService analogdb.KeywordService
	var similarityService analogdb.SimilarityService

	// create service implementations
	postService = postgres.NewPostService(db)
	authorService = postgres.NewAuthorService(db)
	readyService = postgres.NewReadyService(db)
	scrapeService = postgres.NewScrapeService(db)
	keywordService = postgres.NewKeywordService(db)

	// if cache enabled, replace the with cache implementation
	if cfg.App.CacheEnabled {
		postService = redis.NewCachePostService(rdb, postService)
		authorService = redis.NewCacheAuthorService(rdb, authorService)
	}

	similarityService = weaviate.NewSimilarityService(dbVec, postService)

	// if cache enabled, replace the with cache implementation
	if cfg.App.CacheEnabled {
		similarityService = redis.NewCacheSimilarityService(rdb, similarityService)
	}

	server.PostService = postService
	server.AuthorService = authorService
	server.ReadyService = readyService
	server.ScrapeService = scrapeService
	server.KeywordService = keywordService
	server.SimilarityService = similarityService

	if err := server.Run(); err != nil {
		err = fmt.Errorf("Failed to start http server: %w", err)
		fatal(logger, err)
	}

	// wait for shutdown
	<-ctx.Done()
	logger.Info().Msg("Got shutdown signal, starting graceful shutdown")

	if err := server.Close(); err != nil {
		err = fmt.Errorf("Failed to shutdown http server: %w", err)
		fatal(logger, err)
	}

	if err := db.Close(); err != nil {
		err = fmt.Errorf("Failed to shutdown DB: %w", err)
		fatal(logger, err)
	}

	if err := dbVec.Close(); err != nil {
		err = fmt.Errorf("Failed to shutdown vector DB: %w", err)
		fatal(logger, err)
	}

	if err := rdb.Close(); err != nil {
		err = fmt.Errorf("Failed to shutdown redis: %w", err)
		fatal(logger, err)
	}

	if err := metrics.Close(); err != nil {
		err = fmt.Errorf("Failed to shutdown metrics server: %w", err)
		fatal(logger, err)
	}
}

func fatal(logger *logger.Logger, err error) {
	if logger != nil {
		logger.Error().Err(err).Msg("Fatal error, exiting")
	} else {
		err := fmt.Errorf("Fatal error, exiting; error=%w", err)
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(1)

}
