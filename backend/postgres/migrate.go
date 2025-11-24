package postgres

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
)

//go:embed migrations
var migrationFS embed.FS

func (db *DB) migrate() error {
	db.logger.Debug("Start db migrations")
	defer db.logger.Info("Finish db migrations")

	// Check what's specifically in the migrations directory
	migrationsEntries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations directory, err=%w", err)
	}
	db.logger.Debug("Migration files found in embedded filesystem", "count", len(migrationsEntries))
	driver, err := postgres.WithInstance(db.db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("create postgres driver, err=%w", err)
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
			db.logger.Debug("No migrations to apply (already up to date)")
		} else {
			return fmt.Errorf("run migrations, err=%w", err)
		}
	}

	// Check final migration version
	finalVersion, finalDirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		db.logger.Error("Fail get final migration version", "error", err)
	} else if err == migrate.ErrNilVersion {
		db.logger.Debug("No migration version after running migrations")
	} else {
		db.logger.Debug("Final migration state", "version", finalVersion, "dirty", finalDirty)
	}

	db.logger.Debug("Verifying tables exist in database...")

	// List all tables in public schema
	rows, err := db.db.Query(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		ORDER BY table_name
	`)
	if err != nil {
		db.logger.Error("Fail list all tables", "error", err)
	} else {
		defer rows.Close()
		var tableNames []string
		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				db.logger.Error("Fail scan table name", "error", err)
				continue
			}
			tableNames = append(tableNames, tableName)
		}
	}

	db.logger.Info("Complete db migrations")
	return nil
}

func (db *DB) migrateFromPath(migrationsPath string) error {
	db.logger.Debug("Start db migrations from path", "path", migrationsPath)
	defer db.logger.Info("Finish db migrations from path", "path", migrationsPath)

	driver, err := postgres.WithInstance(db.db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("create postgres driver, err=%w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("create migrate instance, err=%w", err)
	}
	defer m.Close()

	// Run all pending migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations, err=%w", err)
	}
	return nil
}
