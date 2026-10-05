package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/clickhouse"
	"github.com/evanofslack/analogdb/config"
	"github.com/evanofslack/analogdb/events"
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
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() { <-c; cancel() }()

	var cfgPath string
	flag.StringVar(&cfgPath, "config", defaultConfigPath, "path to config.yml")
	flag.Parse()

	// generate the config
	cfg, err := config.New(cfgPath)
	if err != nil {
		err = fmt.Errorf("parse app config: %w", err)
		fatal(nil, err)
	}

	// create logger instance
	logger, err := logger.New(cfg.Log.Level, cfg.App.Env, cfg.App.Name)
	if err != nil {
		err = fmt.Errorf("create logger: %w", err)
		fatal(nil, err)
	}
	logger.Info("Initializing application", "version", cfg.App.Version, "env", cfg.App.Env, "loglevel", cfg.Log.Level)

	// initialize otlp tracing
	tracingLogger := logger.WithSubsystem("tracer")
	tracer, err := tracer.New(tracingLogger, cfg)
	if err != nil {
		err = fmt.Errorf("initialize otlp tracing: %w", err)
		fatal(logger, err)
	}

	if cfg.Tracing.Enabled {
		if err := tracer.StartExporter(); err != nil {
			err = fmt.Errorf("start otel exporter: %w", err)
			fatal(logger, err)
		}
	}

	// initialize prometheus metrics
	metricsLogger := logger.WithSubsystem("metrics")
	metrics, err := metrics.New(metricsLogger)
	if err != nil {
		err = fmt.Errorf("initialize prometheus metrics: %w", err)
		fatal(logger, err)
	}

	if cfg.Metrics.Enabled {
		if err := metrics.Serve(cfg.Metrics.Port); err != nil {
			err = fmt.Errorf("start metrics server: %w", err)
			fatal(logger, err)
		}
	}

	// open connection to postgres
	dbLogger := logger.WithSubsystem("database")
	db := postgres.NewDB(cfg.DB.URL, dbLogger, cfg.DB.MigrationEnabled, cfg.DB.MigrationPath, cfg.Tracing.Enabled)
	db.SetPool(postgres.Pool{
		MaxOpenConns:    cfg.DB.MaxOpenConns,
		MaxIdleConns:    cfg.DB.MaxIdleConns,
		ConnMaxLifetime: cfg.DB.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.DB.ConnMaxIdleTime,
	})
	if err := db.Open(); err != nil {
		err = fmt.Errorf("startup database: %w", err)
		fatal(logger, err)
	}
	if err := db.RegisterMetrics(metrics.Registry); err != nil {
		err = fmt.Errorf("register database metrics: %w", err)
		fatal(logger, err)
	}

	// open connection to weaviate
	dbVecLogger := logger.WithSubsystem("vector-database")
	dbVec := weaviate.NewDB(cfg.VectorDB.Host, cfg.VectorDB.Scheme, dbVecLogger, tracer)
	if err := dbVec.Open(); err != nil {
		err = fmt.Errorf("startup vector database: %w", err)
		fatal(logger, err)
	}
	// run weaviate migrations if needed
	if err := dbVec.Migrate(ctx); err != nil {
		err = fmt.Errorf("migrate vector database: %w", err)
		fatal(logger, err)
	}

	// open connection to redis if cache enabled
	var rdb *redis.RDB
	if cfg.App.CacheEnabled {
		redisLogger := logger.WithSubsystem("redis")
		rdb, err = redis.NewRDB(cfg.Redis.URL, redisLogger, metrics, cfg.Tracing.Enabled)
		if err != nil {
			err = fmt.Errorf("startup redis: %w", err)
			fatal(logger, err)
		}
		if err := rdb.Open(); err != nil {
			err = fmt.Errorf("connect to redis: %w", err)
			fatal(logger, err)
		}
	}

	// open connection to kafka if enabled
	var eventService analogdb.EventService
	if cfg.Kafka.Enabled {
		kafkaLogger := logger.WithSubsystem("kafka")
		topic := cfg.Kafka.Topic
		brokers := strings.Split(cfg.Kafka.Brokers, ",")
		opts := events.Options{
			QueueSize:    cfg.Kafka.QueueSize,
			BatchSize:    cfg.Kafka.BatchSize,
			BatchTimeout: cfg.Kafka.BatchTimeout,
		}
		eventService, err = events.New(kafkaLogger, metrics.Registry, topic, brokers, opts)
		if err != nil {
			err = fmt.Errorf("startup kafka: %w", err)
			fatal(logger, err)
		}
	} else {
		eventService = events.NewNoop(logger)
	}

	// open connection to clickhouse if enabled, admin analytics only
	var analytics *clickhouse.DB
	if cfg.ClickHouse.Enabled {
		chLogger := logger.WithSubsystem("clickhouse")
		ch := cfg.ClickHouse
		analytics, err = clickhouse.NewDB(ch.Host, ch.Port, ch.Database, ch.Username, ch.Password, ch.Table, chLogger)
		if err != nil {
			err = fmt.Errorf("startup clickhouse: %w", err)
			fatal(logger, err)
		}
		if err := analytics.Open(); err != nil {
			logger.Error("Fail open clickhouse, admin analytics disabled", "error", err)
			analytics = nil
		}
	}

	// initialize http server
	httpLogger := logger.WithSubsystem("http")
	server := server.New(cfg.HTTP.Port, httpLogger, metrics, cfg)

	// need to clean up this dependency injection
	var postService analogdb.PostService
	var filmService analogdb.FilmService
	var cameraService analogdb.CameraService
	var authorService analogdb.AuthorService
	var readyService analogdb.ReadyService
	var scrapeService analogdb.ScrapeService
	var keywordService analogdb.KeywordService
	var similarityService analogdb.SimilarityService
	var searchService analogdb.SearchService

	// create service implementations
	postService = postgres.NewPostService(db)
	filmService = postgres.NewFilmService(db)
	cameraService = postgres.NewCameraService(db)
	authorService = postgres.NewAuthorService(db)
	readyService = postgres.NewReadyService(db)
	scrapeService = postgres.NewScrapeService(db)
	keywordService = postgres.NewKeywordService(db)

	// if cache enabled, replace the with cache implementation
	if cfg.App.CacheEnabled {
		postService = redis.NewCachePostService(rdb, postService)
		authorService = redis.NewCacheAuthorService(rdb, authorService)
		filmService = redis.NewCacheFilmService(rdb, filmService)
		cameraService = redis.NewCacheCameraService(rdb, cameraService)
		keywordService = redis.NewCacheKeywordService(rdb, keywordService)
	}

	similarityService = weaviate.NewSimilarityService(dbVec, postService)
	searchService = weaviate.NewSearchService(dbVec)

	// if cache enabled, replace the with cache implementation
	if cfg.App.CacheEnabled {
		similarityService = redis.NewCacheSimilarityService(rdb, similarityService, postService)
		searchService = redis.NewCacheSearchService(rdb, searchService)
	}

	server.PostService = postService
	server.FilmService = filmService
	server.CameraService = cameraService
	server.AuthorService = authorService
	server.ReadyService = readyService
	server.ScrapeService = scrapeService
	server.KeywordService = keywordService
	server.SimilarityService = similarityService
	server.SearchService = searchService
	server.EventService = eventService
	server.VectorReadyService = dbVec
	server.VectorCounter = dbVec
	server.VectorLister = dbVec
	server.AdminService = postgres.NewAdminService(db)
	server.ExtractionService = postgres.NewExtractionService(db)
	if analytics != nil {
		server.AnalyticsService = analytics
	}
	if rdb != nil {
		server.CacheReadyService = rdb
	}

	if err := server.Run(); err != nil {
		err = fmt.Errorf("start http server: %w", err)
		fatal(logger, err)
	}

	// wait for shutdown
	<-ctx.Done()
	logger.Info("Got shutdown signal, starting graceful shutdown")

	if err := server.Close(); err != nil {
		logger.Error("Fail shutdown http server", "error", err)
	}

	if err := eventService.Close(); err != nil {
		logger.Error("Fail shutdown event service", "error", err)
	}

	if rdb != nil {
		if err := rdb.Close(); err != nil {
			logger.Error("Fail shutdown redis", "error", err)
		}
	}

	if err := db.Close(); err != nil {
		logger.Error("Fail shutdown DB", "error", err)
	}

	if err := dbVec.Close(); err != nil {
		logger.Error("Fail shutdown vector DB", "error", err)
	}

	if analytics != nil {
		if err := analytics.Close(); err != nil {
			logger.Error("Fail shutdown clickhouse", "error", err)
		}
	}

	if cfg.Metrics.Enabled {
		if err := metrics.Close(); err != nil {
			logger.Error("Fail shutdown metrics server", "error", err)
		}
	}
}

func fatal(logger *logger.Logger, err error) {
	if logger != nil {
		logger.Error("Fatal error, exiting", "error", err)
	} else {
		err := fmt.Errorf("fatal error, exiting; err=%w", err)
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(1)
}
