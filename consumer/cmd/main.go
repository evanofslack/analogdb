package main

import (
	"fmt"
	"log"
	"os"

	"github.com/evanofslack/analogdb-consumer/internal/config"
	"github.com/evanofslack/analogdb-consumer/internal/logging"
)

func main() {
	cfg, err := config.New("config.yaml")
	if err != nil {
		log.Fatal("failed to load config:", err)
	}

	logger, err := logging.New(cfg.Log.Level, cfg.Log.Env, cfg.App.Name)
	if err != nil {
		fmt.Printf("Create logger, err=%v\n", err)
		os.Exit(1)
	}
	logger.Info("starting analogdb-consumer",
		"version", cfg.App.Version,
		"env", cfg.App.Env,
	)

	// TODO: use logger in services
	_ = logger
}
