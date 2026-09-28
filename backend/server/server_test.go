package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/evanofslack/analogdb/config"
	"github.com/evanofslack/analogdb/events"
	"github.com/evanofslack/analogdb/logger"
	"github.com/evanofslack/analogdb/metrics"
	"github.com/joho/godotenv"
)

type mockReady struct{}

func (mr *mockReady) Readyz(ctx context.Context) error {
	return nil
}

func mustOpen(t *testing.T) *Server {
	t.Helper()

	if err := godotenv.Load("../.env"); err != nil {
		fmt.Printf("fail load .env file")
	}

	logger, err := logger.New("debug", "debug", "analogdb-test")
	if err != nil {
		t.Fatal(err)
	}

	metrics, err := metrics.New(logger)
	if err != nil {
		t.Fatal(err)
	}

	config := &config.Config{}

	s := New("0", logger, metrics, config)
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}

	s.ReadyService = &mockReady{}
	s.EventService = events.NewNoop(logger)

	return s
}

func mustClose(t *testing.T, s *Server) {
	t.Helper()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}
