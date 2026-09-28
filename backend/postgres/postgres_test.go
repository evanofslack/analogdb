package postgres

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/evanofslack/analogdb/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testDBName = "testdb"
	testUser   = "testuser"
	testPass   = "testpass"
)

func TestDB(t *testing.T) {
	db, cleanup := mustOpen(t)
	defer cleanup()
	mustClose(t, db)
}

func mustOpen(t *testing.T) (*DB, func()) {
	t.Helper()

	ctx := context.Background()

	// Create PostgreSQL testcontainer
	postgresContainer, err := postgres.Run(ctx,
		"postgres:15",
		postgres.WithDatabase(testDBName),
		postgres.WithUsername(testUser),
		postgres.WithPassword(testPass),
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(60*time.Second),
				wait.ForExposedPort(),
			),
		),
	)
	if err != nil {
		t.Fatalf("Start postgres container, err=%v", err)
	}

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Get connection string, err=%v", err)
	}

	logger, err := logger.New("debug", "debug", "analogdb_test")
	if err != nil {
		t.Fatal(err)
	}

	db := NewDB(connStr, logger, true, "", false)

	if err := db.Open(); err != nil {
		if terminateErr := postgresContainer.Terminate(ctx); terminateErr != nil {
			t.Logf("Terminate container, err=%v", terminateErr)
		}
		t.Fatalf("Open database, err=%v", err)
	}

	cleanup := func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("Terminate postgres container, err=%v", err)
		}
	}

	return db, cleanup
}

func mustClose(t *testing.T, db *DB) {
	t.Helper()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestMustOpen(t *testing.T) {
	db, cleanup := mustOpen(t)
	defer cleanup()

	if db == nil {
		t.Fatal("must return DB")
	}
}

func mustSeed(t *testing.T, db *DB) {
	t.Helper()
	seedSQL, err := os.ReadFile(filepath.Join("testdata", "seed.sql"))
	if err != nil {
		t.Fatalf("Seed DB, err=%v", err)
	}
	if _, err := db.db.Exec(string(seedSQL)); err != nil {
		t.Fatalf("Seed DB, err=%v", err)
	}
}

func mustOpenWithSeed(t *testing.T) (*DB, func()) {
	t.Helper()
	db, cleanup := mustOpen(t)
	mustSeed(t, db)
	return db, cleanup
}

func TestMustOpenWithSeed(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	if db == nil {
		t.Fatal("must return DB")
	}
}

func TestDB_PoolAndMetrics(t *testing.T) {
	db, cleanup := mustOpen(t)
	defer cleanup()

	db.SetPool(Pool{MaxOpenConns: 7, MaxIdleConns: 3, ConnMaxLifetime: time.Minute, ConnMaxIdleTime: time.Second})
	db.applyPool()
	if got := db.db.Stats().MaxOpenConnections; got != 7 {
		t.Errorf("Expected max open connections 7, got %d", got)
	}

	registry := prometheus.NewRegistry()
	if err := db.RegisterMetrics(registry); err != nil {
		t.Fatal(err)
	}
	families, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range families {
		if f.GetName() == "analogdb_go_sql_max_open_connections" {
			found = true
		}
	}
	if !found {
		t.Error("Expected analogdb_go_sql_max_open_connections to be registered")
	}
}
