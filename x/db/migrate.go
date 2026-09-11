package db

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"sync"
)

// ErrUnknownDialect means no dialect package was imported.
var ErrUnknownDialect = errors.New("unknown dialect")

var engines sync.Map // string -> func(*sql.DB, fs.FS, string) error

// Register installs a migrator for Dialect. Dialect packages call this
// from init.
func Register(dialect string, up func(*sql.DB, fs.FS, string) error) {
	engines.Store(dialect, up)
}

func apply(dialect string, conn *sql.DB, fsys fs.FS, dir string) error {
	if dialect == "" {
		return fmt.Errorf("%w: empty", ErrUnknownDialect)
	}
	v, ok := engines.Load(dialect)
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownDialect, dialect)
	}
	up := v.(func(*sql.DB, fs.FS, string) error)
	if err := up(conn, fsys, dir); err != nil {
		return fmt.Errorf("migrate %s: %w", dialect, err)
	}
	return nil
}
