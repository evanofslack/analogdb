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

	cfg, err := config.New(cfgPath)
	if err != nil {
		err = fmt.Errorf("Failed to parse app config: %w", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	logger, err := logger.New(cfg.Log.Level, cfg.App.Env)
	if err != nil {
		err = fmt.Errorf("Failed to create logger: %w", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	logger.Info().Str("App", cfg.App.Name).Str("Version", cfg.App.Version).Msg("Initializing application")

	if webhookURL := cfg.Log.WebhookURL; webhookURL != "" {
		logger = logger.WithSlackNotifier(webhookURL)
	}

	dbLogger := logger.WithService("database")
	db := postgres.NewDB(cfg.DB.URL, dbLogger)
	if err := db.Open(); err != nil {
		err = fmt.Errorf("Failed to startup datebase: %w", err)
		logger.Err(err).Msg("Fatal error, exiting")
		os.Exit(1)
	}

	// open connection to weaviate
	dbVecLogger := logger.WithService("vector-database")
	dbVec := weaviate.NewDB(cfg.VectorDB.Host, cfg.VectorDB.Scheme, dbVecLogger)
	if err := dbVec.Open(); err != nil {
		err = fmt.Errorf("Failed to startup vector datebase: %w", err)
		logger.Err(err).Msg("Fatal error, exiting")
		os.Exit(1)
	}
	// run weaviate migrations if needed
	// creates the schema if it does not exist
	if err := dbVec.Migrate(ctx); err != nil {
		err = fmt.Errorf("Failed to migrate vector datebase: %w", err)
		logger.Err(err).Msg("Fatal error, exiting")
		os.Exit(1)
	}

	postService := postgres.NewPostService(db)

	httpLogger := logger.WithService("http")
	server := server.New(cfg.HTTP.Port, httpLogger)
	server.PostService = postService
	server.ReadyService = postgres.NewReadyService(db)
	server.AuthorService = postgres.NewAuthorService(db)
	server.ScrapeService = postgres.NewScrapeService(db)
	server.SimilarityService = weaviate.NewSimilarityService(dbVec, postService)
	if err := server.Run(); err != nil {
		err = fmt.Errorf("Failed to start http server: %w", err)
		logger.Err(err).Msg("Fatal error, exiting")
		os.Exit(1)
	}

	<-ctx.Done()
	logger.Info().Msg("Got shutdown signal, starting graceful shutdown")

	if err := server.Close(); err != nil {
		err = fmt.Errorf("Failed to shutdown http server: %w", err)
		logger.Err(err).Msg("Fatal error, exiting")
		os.Exit(1)
	}

	if err := db.Close(); err != nil {
		err = fmt.Errorf("Failed to shutdown DB: %w", err)
		logger.Err(err).Msg("Fatal error, exiting")
		os.Exit(1)
	}
}