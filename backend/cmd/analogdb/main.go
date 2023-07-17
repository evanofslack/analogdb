package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/evanofslack/analogdb/config"
	"github.com/evanofslack/analogdb/logger"
	"github.com/evanofslack/analogdb/postgres"
	"github.com/evanofslack/analogdb/server"
	"github.com/evanofslack/analogdb/weaviate"
	"github.com/evanofslack/analogdb/metrics"
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// create logger instance
	logger, err := logger.New(cfg.Log.Level, cfg.App.Env)
	if err != nil {
		err = fmt.Errorf("Failed to create logger: %w", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	logger.Info().Str("App", cfg.App.Name).Str("Version", cfg.App.Version).Str("env", cfg.App.Env).Str("loglevel", cfg.Log.Level).Msg("Initializing application")

	// add slack webhook to logger to notify on error
	if webhookURL := cfg.Log.WebhookURL; webhookURL != "" && cfg.App.Env != "debug" {
		logger = logger.WithSlackNotifier(webhookURL)
	}

	// open connection to postgres
	dbLogger := logger.WithService("database")
	db := postgres.NewDB(cfg.DB.URL, dbLogger)
	if err := db.Open(); err != nil {
		err = fmt.Errorf("Failed to startup datebase: %w", err)
		logger.Error().Err(err).Msg("Fatal error, exiting")
		os.Exit(1)
	}

	// open connection to weaviate
	dbVecLogger := logger.WithService("vector-database")
	dbVec := weaviate.NewDB(cfg.VectorDB.Host, cfg.VectorDB.Scheme, dbVecLogger)
	if err := dbVec.Open(); err != nil {
		err = fmt.Errorf("Failed to startup vector datebase: %w", err)
		logger.Error().Err(err).Msg("Fatal error, exiting")
		os.Exit(1)
	}
	// run weaviate migrations if needed
	// creates the schema if it does not exist
	if err := dbVec.Migrate(ctx); err != nil {
		err = fmt.Errorf("Failed to migrate vector datebase: %w", err)
		logger.Error().Err(err).Msg("Fatal error, exiting")
		os.Exit(1)
	}

	// initialize prometheus metrics
	metricsLogger := logger.WithService("metrics")
	metrics, err := metrics.New(metricsLogger)
	if err != nil {
		err = fmt.Errorf("Failed to initialize prometheus metrics: %w", err)
		logger.Error().Err(err).Msg("Fatal error, exiting")
		os.Exit(1)
	}

	// initialize http server
	httpLogger := logger.WithService("http")
	server := server.New(cfg.HTTP.Port, httpLogger, metrics)

	postService := postgres.NewPostService(db)

	server.PostService = postService
	server.ReadyService = postgres.NewReadyService(db)
	server.AuthorService = postgres.NewAuthorService(db)
	server.ScrapeService = postgres.NewScrapeService(db)
	server.SimilarityService = weaviate.NewSimilarityService(dbVec, postService)
	if err := server.Run(); err != nil {
		err = fmt.Errorf("Failed to start http server: %w", err)
		logger.Error().Err(err).Msg("Fatal error, exiting")
		os.Exit(1)
	}

	// wait for shutdown
	<-ctx.Done()
	logger.Info().Msg("Got shutdown signal, starting graceful shutdown")

	if err := server.Close(); err != nil {
		err = fmt.Errorf("Failed to shutdown http server: %w", err)
		logger.Error().Err(err).Msg("Fatal error, exiting")
		os.Exit(1)
	}

	if err := db.Close(); err != nil {
		err = fmt.Errorf("Failed to shutdown DB: %w", err)
		logger.Error().Err(err).Msg("Fatal error, exiting")
		os.Exit(1)
	}

	if err := dbVec.Close(); err != nil {
		err = fmt.Errorf("Failed to shutdown vector DB: %w", err)
		logger.Error().Err(err).Msg("Fatal error, exiting")
		os.Exit(1)
	}
}
