// Package postgres registers the postgres URL schemes (pgx stdlib).
//
//	import _ "github.com/lewtec/lewkit/x/db/postgres"
//
// Schemes: postgres, postgresql.
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

	_ "github.com/jackc/pgx/v5/stdlib"
)

func init() {
	c := db.Connector{Driver: "pgx", Up: up}
	db.Register("postgres", c)
	db.Register("postgresql", c)
}

func up(conn *sql.DB, fsys fs.FS) error {
	src, err := iofs.New(fsys, ".")
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
