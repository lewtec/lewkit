// Package sqlite registers the sqlite URL schemes.
//
//	import _ "github.com/lewtec/lewkit/x/db/sqlite"
//
// Schemes: sqlite, sqlite3, file. Bare paths and :memory: also map here.
package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	migsqlite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/lewtec/lewkit/x/db"

	_ "modernc.org/sqlite"
)

func init() {
	c := db.Connector{Driver: "sqlite", Up: up}
	db.Register("sqlite", c)
	db.Register("sqlite3", c)
	db.Register("file", c)
}

func up(conn *sql.DB, fsys fs.FS) error {
	src, err := iofs.New(fsys, ".")
	if err != nil {
		return fmt.Errorf("iofs: %w", err)
	}
	inst, err := migsqlite.WithInstance(conn, &migsqlite.Config{})
	if err != nil {
		return fmt.Errorf("instance: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "sqlite3", inst)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
