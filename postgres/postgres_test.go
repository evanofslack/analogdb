package postgres

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestDB(t *testing.T) {
	db := mustOpen(t)
	mustClose(t, db)
}

func mustOpen(t *testing.T) *DB {
	t.Helper()

	if err := godotenv.Load("../.env"); err != nil {
		t.Error("Error loading .env file")
	}

	// connect to local db for testing
	dsn := os.Getenv("POSTGRES_DATABASE_URL")

	db := NewDB(dsn)

	if err := db.Open(); err != nil {
		t.Fatal(err)
	}
	return db
}

func mustClose(t *testing.T, db *DB) {
	t.Helper()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestMustOpen(t *testing.T) {
	db := mustOpen(t)
	if db == nil {
		t.Fatal("must return DB")
	}
}
