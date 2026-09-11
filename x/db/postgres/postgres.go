// Package postgres registers the postgres migrator and the default
// pgx database/sql driver name.
package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	migpostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/lewtec/lewkit/x/db"
)

// Engine is the Source methods for pgx stdlib + golang-migrate postgres.
type Engine struct{}

// Driver is the database/sql name registered by pgx/stdlib.
func (Engine) Driver() string { return "pgx" }

// Dialect is golang-migrate's postgres database name.
func (Engine) Dialect() string { return "postgres" }

func init() {
	db.Register("postgres", up)
}

func up(conn *sql.DB, fsys fs.FS, dir string) error {
	src, err := iofs.New(fsys, dir)
	if err != nil {
		return fmt.Errorf("iofs: %w", err)
	}
	inst, err := migpostgres.WithInstance(conn, &migpostgres.Config{})
	if err != nil {
		return fmt.Errorf("instance: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", inst)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
