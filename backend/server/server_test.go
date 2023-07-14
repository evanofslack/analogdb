package server

import (
	"os"
	"testing"

	"github.com/evanofslack/analogdb/postgres"
	"github.com/evanofslack/analogdb/logger"
	"github.com/joho/godotenv"
)

func mustOpen(t *testing.T) (*Server, *postgres.DB) {
	t.Helper()

	if err := godotenv.Load("../.env"); err != nil {
		t.Error("Error loading .env file")
	}

	// httpserver test currently require DB, can be mocked out instead
	dsn := os.Getenv("POSTGRES_DATABASE_URL")

	logger, err := logger.New("debug", "debug")
	if err != nil {
		t.Fatal(err)

	}
	db := postgres.NewDB(dsn, logger)
	if err := db.Open(); err != nil {
		t.Fatal(err)
	}

	ps := postgres.NewPostService(db)
	rs := postgres.NewReadyService(db)
	as := postgres.NewAuthorService(db)

	s := New("8080", logger)
	s.PostService = ps
	s.ReadyService = rs
	s.AuthorService = as
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	return s, db
}

func mustClose(t *testing.T, s *Server, db *postgres.DB) {
	t.Helper()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}
