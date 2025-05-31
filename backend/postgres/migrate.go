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
	db.logger.Debug().Msg("Starting DB migrations")

	// Check what's specifically in the migrations directory
	migrationsEntries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		db.logger.Error().Err(err).Msg("Failed to read migrations directory from embedded filesystem")
		return fmt.Errorf("read migrations directory, err=%w", err)
	}
	db.logger.Debug().Int("count", len(migrationsEntries)).Msg("Migration files found in embedded filesystem")
	for i, entry := range migrationsEntries {
		db.logger.Debug().
			Str("name", entry.Name()).
			Msg("Migration file")
	}

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
			db.logger.Debug().Msg("No migrations to apply (already up to date)")
		} else {
			return fmt.Errorf("run migrations, err=%w", err)
		}
	}

	// Check final migration version
	finalVersion, finalDirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		db.logger.Error().Err(err).Msg("Failed to get final migration version")
	} else if err == migrate.ErrNilVersion {
		db.logger.Debug().Msg("Still no migration version after running migrations")
	} else {
		db.logger.Debug().
			Uint("version", finalVersion).
			Bool("dirty", finalDirty).
			Msg("Final migration state")
	}

	db.logger.Debug().Msg("Verifying tables exist in database...")

	// List all tables in public schema
	rows, err := db.db.Query(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		ORDER BY table_name
	`)
	if err != nil {
		db.logger.Error().Err(err).Msg("Failed to list all tables")
	} else {
		defer rows.Close()
		var tableNames []string
		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				db.logger.Error().Err(err).Msg("Failed to scan table name")
				continue
			}
			tableNames = append(tableNames, tableName)
		}
		db.logger.Debug().Strs("tables", tableNames).Int("count", len(tableNames)).Msg("All tables in public schema")
	}

	db.logger.Info().Msg("Completed DB migrations")
	return nil
}

func (db *DB) migrateFromPath(migrationsPath string) error {
	db.logger.Debug().Str("path", migrationsPath).Msg("Starting DB migrations from path")

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

	db.logger.Info().Str("path", migrationsPath).Msg("Completed DB migrations from path")
	return nil
}
