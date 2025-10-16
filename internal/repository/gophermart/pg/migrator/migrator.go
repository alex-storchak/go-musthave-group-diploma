package migrator

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func ApplyMigrations(db *sql.DB, dirName string) error {

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("unable to create db instance: %v", err)
	}

	migrator, err := migrate.NewWithDatabaseInstance(dirName, "pgx", driver)
	if err != nil {
		return fmt.Errorf("unable to create migration: %v", err)
	}

	if err = migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("unable to apply migrations %v", err)
	}

	return nil
}
