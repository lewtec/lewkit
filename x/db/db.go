// Package db is a cmd product that opens a database, runs embedded
// migrations, and constructs sqlc queries.
//
// SQL is one URL token. The command field supplies the flag name, so
// one spec can hold a postgres field and a sqlite field without a
// colliding --database tag.
//
//	type serve struct {
//		PG   db.SQL[pg.Queries, pgMig]   `long:"postgres" help:"postgres URL"`
//		Lite db.SQL[lite.Queries, liteMig] `long:"sqlite" help:"sqlite path"`
//	}
//
// Source is a zero-value type: embed.FS is a value, so it cannot be a
// type parameter. The app binds the folder with methods, the same way
// goose SetBaseFS and golang-migrate iofs.New take an embed.FS.
//
//	type liteMig struct{ sqlite.Engine }
//	func (liteMig) FS() fs.FS   { return liteFS }
//	func (liteMig) Dir() string { return "migrations" }
//
// Import the dialect package so its migrator is registered:
//
//	import _ "github.com/lewtec/lewkit/x/db/sqlite"
//
// sqlc emits New(db DBTX) *Queries per engine. OpenStdlib passes *sql.DB
// to that constructor. Open is for pgx pools: migrate on database/sql,
// then connect with the native driver, as ciborg does for postgres.
//
// Two engines in one process is two SQL fields, or one URL plus a driver
// switch that calls FromURL. sqlc itself is one generated package per
// engine (ciborg's sqlc.yaml).
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/lewtec/lewkit/x/cmd"
)

// Source is the zero-value migration folder and engine for SQL.
type Source interface {
	FS() fs.FS
	Dir() string
	Driver() string
	Dialect() string
}

// SQL is a cmd product: one URL token. Tag the field at the embed site.
type SQL[Q any, S Source] struct {
	URL cmd.StringArg
}

// Handle is an open database and its sqlc query type.
// DB is set by OpenStdlib and is nil after Open.
type Handle[Q any] struct {
	DB    *sql.DB
	Q     Q
	close func() error
}

// ErrEphemeral is returned when Migrate or Open is used with an
// in-memory URL. Those need one *sql.DB; use OpenStdlib.
var ErrEphemeral = errors.New("in-memory database requires OpenStdlib")

// FromURL fills SQL from a DSN already in hand (env, driver switch).
func FromURL[Q any, S Source](url string) SQL[Q, S] {
	var s SQL[Q, S]
	_ = s.URL.Parse(url)
	return s
}

// OpenStdlib opens database/sql, migrates, and builds Q from *sql.DB.
func (s SQL[Q, S]) OpenStdlib(ctx context.Context, newQ func(*sql.DB) Q) (*Handle[Q], error) {
	conn, err := s.openDB(ctx)
	if err != nil {
		return nil, err
	}
	var src S
	if err := apply(src.Dialect(), conn, src.FS(), src.Dir()); err != nil {
		return nil, errors.Join(err, conn.Close())
	}
	return &Handle[Q]{DB: conn, Q: newQ(conn)}, nil
}

// Open migrates on a short-lived database/sql connection, then calls
// connect with the URL. Use this when Q is built from a pgx pool.
func (s SQL[Q, S]) Open(ctx context.Context, connect func(context.Context, string) (Q, func() error, error)) (*Handle[Q], error) {
	if err := s.Migrate(ctx); err != nil {
		return nil, err
	}
	q, closer, err := connect(ctx, s.URL.Value())
	if err != nil {
		return nil, err
	}
	return &Handle[Q]{Q: q, close: closer}, nil
}

// Migrate opens database/sql, runs pending migrations, and closes.
func (s SQL[Q, S]) Migrate(ctx context.Context) error {
	if ephemeral(s.URL.Value()) {
		return ErrEphemeral
	}
	conn, err := s.openDB(ctx)
	if err != nil {
		return err
	}
	var src S
	return errors.Join(apply(src.Dialect(), conn, src.FS(), src.Dir()), conn.Close())
}

// Close closes the handle.
func (h *Handle[Q]) Close() error {
	if h == nil {
		return nil
	}
	if h.close != nil {
		return h.close()
	}
	if h.DB != nil {
		return h.DB.Close()
	}
	return nil
}

func (s SQL[Q, S]) openDB(ctx context.Context) (*sql.DB, error) {
	var src S
	driver := src.Driver()
	conn, err := sql.Open(driver, s.URL.Value())
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", driver, err)
	}
	if err := conn.PingContext(ctx); err != nil {
		return nil, errors.Join(fmt.Errorf("ping %s: %w", driver, err), conn.Close())
	}
	return conn, nil
}

func ephemeral(url string) bool {
	if url == ":memory:" || strings.HasPrefix(url, "file::memory:") {
		return true
	}
	_, after, ok := strings.Cut(url, "?")
	return ok && strings.Contains(after, "mode=memory")
}
