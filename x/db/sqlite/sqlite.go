// Package sqlite registers the sqlite URL schemes.
//
//	import _ "github.com/lewtec/lewkit/x/db/sqlite"
//
// Schemes: sqlite, sqlite3, file. Bare paths and :memory: also map here.
package sqlite

import (
	"database/sql"
	"io/fs"

	"github.com/golang-migrate/migrate/v4/database"
	migsqlite "github.com/golang-migrate/migrate/v4/database/sqlite"

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
	return db.Up(conn, fsys, "sqlite3", func(c *sql.DB) (database.Driver, error) {
		return migsqlite.WithInstance(c, &migsqlite.Config{})
	})
}
