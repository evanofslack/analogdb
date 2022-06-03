package server

import (
	"fmt"
	"os"
	"testing"

	"github.com/evanofslack/analogdb/postgres"
	"github.com/joho/godotenv"
)

func mustOpen(t *testing.T) *Server {
	t.Helper()

	if err := godotenv.Load("../.env"); err != nil {
		t.Error("Error loading .env file")
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		os.Getenv("DBHOST"), os.Getenv("DBPORT"), os.Getenv("DBUSER"),
		os.Getenv("DBPASSWORD"), os.Getenv("DBNAME"))

	db := postgres.NewDB(dsn)
	if err := db.Open(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	ps := postgres.NewPostService(db)

	s := New()
	s.PostService = ps
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	return s
}

func mustClose(t *testing.T, s *Server) {
	t.Helper()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}
