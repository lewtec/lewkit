package db

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Up applies iofs migrations at "." on an open connection.
// name is the golang-migrate database instance name (postgres, sqlite3, …).
func Up(conn *sql.DB, fsys fs.FS, name string, instance func(*sql.DB) (database.Driver, error)) error {
	src, err := iofs.New(fsys, ".")
	if err != nil {
		return fmt.Errorf("iofs: %w", err)
	}
	inst, err := instance(conn)
	if err != nil {
		return fmt.Errorf("instance: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, name, inst)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
