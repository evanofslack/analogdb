package postgres

import (
	"embed"
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
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations, err=%w", err)
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
