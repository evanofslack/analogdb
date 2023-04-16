package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/evanofslack/analogdb/config"
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	db := postgres.NewDB(cfg.DB.URL)
	if err := db.Open(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// open connection to weaviate
	dbVec := weaviate.NewDB(cfg.VectorDB.Host, cfg.VectorDB.Scheme)
	if err := dbVec.Open(); err != nil {
		fmt.Println("failed to open dbVec")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// run migrations if needed
	// creates the schema if it does not exist
	if err := dbVec.Migrate(ctx); err != nil {
		fmt.Println("failed to migrate dbVec")
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

	// temp test
	// allIDs, err := server.PostService.AllPostIDs(ctx)
	// if err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// 	os.Exit(1)
	// }

	// err = server.SimilarityService.BatchEncodePosts(ctx, allIDs, 100)
	// if err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// 	os.Exit(1)
	// }

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
