// Package postgres registers the postgres URL schemes (pgx stdlib).
//
//	import _ "github.com/lewtec/lewkit/x/db/postgres"
//
// Schemes: postgres, postgresql.
package postgres

import (
	"database/sql"
	"io/fs"

	"github.com/golang-migrate/migrate/v4/database"
	migpostgres "github.com/golang-migrate/migrate/v4/database/postgres"

	"github.com/lewtec/lewkit/x/db"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func init() {
	c := db.Connector{Driver: "pgx", Up: up}
	db.Register("postgres", c)
	db.Register("postgresql", c)
}

func up(conn *sql.DB, fsys fs.FS) error {
	return db.Up(conn, fsys, "postgres", func(c *sql.DB) (database.Driver, error) {
		return migpostgres.WithInstance(c, &migpostgres.Config{})
	})
}
