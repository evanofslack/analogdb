package server

import (
	"fmt"
	"os"
	"testing"

	"github.com/evanofslack/analogdb/postgres"
	"github.com/joho/godotenv"
)

func mustOpen(t *testing.T) (*Server, *postgres.DB) {
	t.Helper()

	if err := godotenv.Load("../.env"); err != nil {
		t.Error("Error loading .env file")
	}

	// httpserver test currently require DB, can be mocked out instead
	dsn := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		os.Getenv("DBHOST"), os.Getenv("DBPORT"), os.Getenv("DBUSER"),
		os.Getenv("DBPASSWORD"), os.Getenv("DBNAME"))

	db := postgres.NewDB(dsn)
	if err := db.Open(); err != nil {
		t.Fatal(err)
	}

	ps := postgres.NewPostService(db)
	rs := postgres.NewReadyService(db)

	s := New("8080")
	s.PostService = ps
	s.ReadyService = rs
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
