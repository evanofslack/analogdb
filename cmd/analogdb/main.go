package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/evanofslack/analogdb/config"
	"github.com/evanofslack/analogdb/postgres"
	"github.com/evanofslack/analogdb/server"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	cfgPath := "config.yml"
	cfg, err := config.New(cfgPath)
	if err != nil {
		log.Fatal(err)
	}

	db := postgres.NewDB(cfg.DB.URL)
	if err := db.Open(); err != nil {
		log.Fatal(err)
	}

	server := server.New(cfg.HTTP.Port)
	server.PostService = postgres.NewPostService(db)
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()

}
