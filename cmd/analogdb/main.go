package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/joho/godotenv"

	"github.com/evanofslack/analogdb/config"
	"github.com/evanofslack/analogdb/postgres"
	"github.com/evanofslack/analogdb/server"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	if err := godotenv.Load("../../.env"); err != nil {
		log.Fatal("Error loading .env file")
	}

	cfgPath := "../../config/config.yml"
	cfg, err := config.New(cfgPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(cfg.DB.URL)
	db := postgres.NewDB(cfg.DB.URL)
	if err := db.Open(); err != nil {
		log.Fatal(err)
	}

	ps := postgres.NewPostService(db)

	server := server.New(cfg.HTTP.Port)
	server.PostService = ps
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("running on port ", cfg.HTTP.Port)

	<-ctx.Done()

}
