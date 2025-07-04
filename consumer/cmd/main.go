package main

import (
	"log"

	"github.com/evanofslack/analogdb-consumer/internal/config"
)

func main() {
	cfg, err := config.New("config.yaml")
	if err != nil {
		log.Fatal("failed to load config:", err)
	}
	
	_ = cfg
}
