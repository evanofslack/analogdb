package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/evanofslack/analogdb/config"
	"github.com/evanofslack/analogdb/postgres"
	"github.com/evanofslack/analogdb/server"
	"github.com/evanofslack/analogdb/weaviate"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	cfgPath := "config.yml"
	cfg, err := config.New(cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	db := postgres.NewDB(cfg.DB.URL)
	if err := db.Open(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	dbVec := weaviate.NewDB(cfg.VectorDB.Host, cfg.VectorDB.Scheme)
	if err := db.Open(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	postService := postgres.NewPostService(db)

	server := server.New(cfg.HTTP.Port)
	server.PostService = postService
	server.ReadyService = postgres.NewReadyService(db)
	server.AuthorService = postgres.NewAuthorService(db)
	server.ScrapeService = postgres.NewScrapeService(db)
	server.SimilarityService = weaviate.NewSimilarityService(dbVec, postService)
	if err := server.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	<-ctx.Done()

	if err := server.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := db.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
