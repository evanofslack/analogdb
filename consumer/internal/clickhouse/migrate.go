package clickhouse

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	clickhouse_migrate "github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations
var migrationFS embed.FS

func (c *Client) migrate() error {
	c.logger.Debug("Starting db migrations")

	// Check what's specifically in the migrations directory
	migrationsEntries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		c.logger.Error("Read migrations directory from embedded filesystem", "error", err)
		return fmt.Errorf("read migrations directory, err=%w", err)
	}
	c.logger.Debug("Migration files found in embedded filesystem", "count", len(migrationsEntries))
	for _, entry := range migrationsEntries {
		c.logger.Debug("Migration file", "file", entry)
	}

	db, err := sql.Open("clickhouse", c.dsn)
	if err != nil {
		return fmt.Errorf("open db connection: %w", err)
	}
	defer c.Close()

	driver, err := clickhouse_migrate.WithInstance(db, &clickhouse_migrate.Config{})
	if err != nil {
		return fmt.Errorf("create migrate driver: %w", err)
	}

	// Create filesystem source from embedded files
	migrationsFS, err := fs.Sub(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("create migrations filesystem, err=%w", err)
	}

	source, err := iofs.New(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("create migrations source, err=%w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("create migrate instance, err=%w", err)
	}

	// Run all pending migrations
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			c.logger.Debug("No migrations to apply (already up to date)")
		} else {
			return fmt.Errorf("run migrations, err=%w", err)
		}
	}

	// Check final migration version
	finalVersion, finalDirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		c.logger.Error("Failed to get final migration version", "error", err)
	} else if err == migrate.ErrNilVersion {
		c.logger.Debug("Still no migration version after running migrations")
	} else {
		c.logger.Debug("Final migration state", "version", finalVersion, "dirty", finalDirty)
	}

	c.logger.Info("Completed db migrations")
	return nil
}

func (c *Client) migrateFromPath(path string) error {
	c.logger.Debug("Start run migrations", "path", path)
	db, err := sql.Open("clickhouse", c.dsn)
	if err != nil {
		return fmt.Errorf("open c connection: %w", err)
	}
	defer c.Close()

	driver, err := clickhouse_migrate.WithInstance(db, &clickhouse_migrate.Config{})
	if err != nil {
		return fmt.Errorf("create migrate driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", path),
		"clickhouse",
		driver,
	)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	c.logger.Info("Finish run migrations", "path", path)

	return nil
}
