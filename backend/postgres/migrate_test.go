package postgres

import (
	"io/fs"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func mustMigrate(t *testing.T, db *DB) *migrate.Migrate {
	t.Helper()
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
	return m
}

// migrationBefore returns the version of the migration preceding the named one
func migrationBefore(t *testing.T, name string) uint {
	t.Helper()
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	var version uint64
	for _, e := range entries {
		prefix, found := strings.CutSuffix(e.Name(), "_"+name+".up.sql")
		if !found {
			continue
		}
		if version, err = strconv.ParseUint(prefix, 10, 64); err != nil {
			t.Fatal(err)
		}
	}
	if version == 0 {
		t.Fatalf("No migration named %s", name)
	}
	migrationsFS, err := fs.Sub(migrationFS, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	source, err := iofs.New(migrationsFS, ".")
	if err != nil {
		t.Fatal(err)
	}
	prev, err := source.Prev(uint(version))
	if err != nil {
		t.Fatalf("No migration before %s, err=%v", name, err)
	}
	return prev
}

func TestMigrateUpDownUp(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	m := mustMigrate(t, db)

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

	timestampColumns := func() int {
		t.Helper()
		var count int
		query := `SELECT count(*) FROM information_schema.columns WHERE table_name = 'pictures' AND column_name IN ('created', 'updated')`
		if err := db.db.QueryRow(query).Scan(&count); err != nil {
			t.Fatal(err)
		}
		return count
	}
	if got := timestampColumns(); got != 2 {
		t.Fatalf("Expected created and updated on pictures after up, got %d", got)
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
	if got := timestampColumns(); got != 0 {
		t.Fatalf("Expected no created or updated on pictures after down, got %d", got)
	}

	if err := m.Up(); err != nil {
		t.Fatalf("Migrate up again, err=%v", err)
	}
	if got := indexCount(); got != 10 {
		t.Fatalf("Expected 10 indexes after second up, got %d", got)
	}
	if got := timestampColumns(); got != 2 {
		t.Fatalf("Expected created and updated on pictures after second up, got %d", got)
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
	if got := timestampColumns(); got != 2 {
		t.Fatalf("Expected created and updated on pictures after full down and up, got %d", got)
	}
}

func TestMigratePostTimestampsBackfill(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	m := mustMigrate(t, db)
	before := migrationBefore(t, "add_post_timestamps")
	if err := m.Migrate(before); err != nil {
		t.Fatalf("Migrate down to %d, err=%v", before, err)
	}

	// a post with no post_updates rows
	var bare int
	query := `INSERT INTO pictures (url, permalink, time) VALUES ('https://example.com/bare.jpg', 'reddit.com/bare', 1641600000) RETURNING id`
	if err := db.db.QueryRow(query).Scan(&bare); err != nil {
		t.Fatal(err)
	}

	if err := m.Up(); err != nil {
		t.Fatalf("Migrate up, err=%v", err)
	}

	// seed.sql post_updates: post 1 last patched at 1641254400, post 2 at 1641340800
	cases := []struct {
		id      int
		created int64
		updated int64
	}{
		{1, 1640995200, 1641254400},
		{2, 1641081600, 1641340800},
		{bare, 1641600000, 1641600000},
	}
	for _, c := range cases {
		var created, updated time.Time
		if err := db.db.QueryRow("SELECT created, updated FROM pictures WHERE id = $1", c.id).Scan(&created, &updated); err != nil {
			t.Fatal(err)
		}
		if created.Unix() != c.created || updated.Unix() != c.updated {
			t.Errorf("Post %d: expected created %d updated %d, got %d %d", c.id, c.created, c.updated, created.Unix(), updated.Unix())
		}
	}

	var nulls int
	if err := db.db.QueryRow("SELECT count(*) FROM pictures WHERE created IS NULL OR updated IS NULL OR updated < created").Scan(&nulls); err != nil {
		t.Fatal(err)
	}
	if nulls != 0 {
		t.Errorf("Expected every post to have created <= updated, got %d bad rows", nulls)
	}
}
