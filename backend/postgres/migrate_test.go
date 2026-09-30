package postgres

import (
	"io/fs"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func TestMigrateUpDownUp(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	migrationsFS, err := fs.Sub(migrationFS, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	source, err := iofs.New(migrationsFS, ".")
	if err != nil {
		t.Fatal(err)
	}
	driver, err := postgres.WithInstance(db.db, &postgres.Config{})
	if err != nil {
		t.Fatal(err)
	}
	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		t.Fatal(err)
	}

	indexCount := func() int {
		t.Helper()
		var count int
		query := `SELECT count(*) FROM pg_indexes WHERE schemaname = 'public' AND indexname = ANY($1::text[])`
		names := "{idx_pictures_time_id,idx_pictures_score_id,idx_pictures_author,idx_pictures_camera,idx_pictures_film,idx_keywords_post_id,idx_keywords_word,idx_colors_post_id,idx_colors_html,idx_post_updates_post}"
		if err := db.db.QueryRow(query, names).Scan(&count); err != nil {
			t.Fatal(err)
		}
		return count
	}

	if got := indexCount(); got != 10 {
		t.Fatalf("Expected 10 indexes after up, got %d", got)
	}

	tableExists := func(name string) bool {
		t.Helper()
		var exists bool
		if err := db.db.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, name).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		return exists
	}
	if !tableExists("post_extractions") {
		t.Fatal("Expected post_extractions after up")
	}

	if err := m.Migrate(8); err != nil {
		t.Fatalf("Migrate down to version 8, err=%v", err)
	}
	if got := indexCount(); got != 0 {
		t.Fatalf("Expected 0 indexes after down, got %d", got)
	}
	if tableExists("post_extractions") {
		t.Fatal("Expected no post_extractions after down")
	}

	if err := m.Up(); err != nil {
		t.Fatalf("Migrate up again, err=%v", err)
	}
	if got := indexCount(); got != 10 {
		t.Fatalf("Expected 10 indexes after second up, got %d", got)
	}

	if err := m.Down(); err != nil {
		t.Fatalf("Migrate all the way down, err=%v", err)
	}
	if err := m.Up(); err != nil {
		t.Fatalf("Migrate all the way up, err=%v", err)
	}
	if got := indexCount(); got != 10 {
		t.Fatalf("Expected 10 indexes after full down and up, got %d", got)
	}
}
