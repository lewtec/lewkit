// Package sqlite registers the sqlite3 migrator and the default
// modernc.org/sqlite driver name.
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
)

// Engine is the Source methods for modernc sqlite + golang-migrate sqlite3.
type Engine struct{}

// Driver is the database/sql name registered by modernc.org/sqlite.
func (Engine) Driver() string { return "sqlite" }

// Dialect is golang-migrate's sqlite3 database name.
func (Engine) Dialect() string { return "sqlite3" }

func init() {
	db.Register("sqlite3", up)
}

func up(conn *sql.DB, fsys fs.FS, dir string) error {
	src, err := iofs.New(fsys, dir)
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
