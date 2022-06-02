package server

import (
	"fmt"
	"log"
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
		log.Fatal(err)
	}

	ps := postgres.NewPostService(db)

	s := New()
	s.PostService = ps
	s.Run()
	return s
}

func mustClose(t *testing.T, s *Server) {
	t.Helper()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}
