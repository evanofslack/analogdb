package server

import (
	"context"
	"testing"

	"github.com/evanofslack/analogdb/config"
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
		t.Error("Error loading .env file")
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

	s := New("8080", logger, metrics, config)
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}

	s.ReadyService = &mockReady{}

	return s
}

func mustClose(t *testing.T, s *Server) {
	t.Helper()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}
