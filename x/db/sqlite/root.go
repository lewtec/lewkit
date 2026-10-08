// Package sqlite registers the sqlite URL schemes.
//
//	import _ "github.com/lewtec/lewkit/x/db/sqlite"
//
// sqlite, sqlite3, and file URLs, plus a bare path and :memory:,
// canonicalize to this connector.
package sqlite

import (
	"context"
	"database/sql"
	"io/fs"

	"github.com/golang-migrate/migrate/v4/database"
	migsqlite "github.com/golang-migrate/migrate/v4/database/sqlite"

	"github.com/lewtec/lewkit/x/db"

	_ "modernc.org/sqlite"
)

func init() {
	db.Register("sqlite", db.Connector{Driver: "sqlite", Up: up})
}

func up(ctx context.Context, conn *sql.DB, fsys fs.FS) error {
	return db.Up(ctx, conn, fsys, "sqlite3", func(c *sql.DB) (database.Driver, error) {
		return migsqlite.WithInstance(c, &migsqlite.Config{})
	})
}
