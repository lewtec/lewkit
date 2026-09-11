// Package db is a --database URL flag. Value() migrates and runs sqlc queries.
//
//	type serve struct {
//		DB db.Arg[Querier] `long:"database"`
//	}
//
//	func (s *serve) Run(ctx context.Context) error {
//		d := s.DB.Value()
//		if err := d.Open(ctx, migrations, sqlc.New); err != nil {
//			return err
//		}
//		defer d.Close()
//		return d.Tx(ctx, func(q Querier) error {
//			return q.Insert(ctx, ...)
//		})
//	}
//
// Blank-import engines so their schemes register:
//
//	import _ "github.com/lewtec/lewkit/x/db/sqlite"
//	import _ "github.com/lewtec/lewkit/x/db/postgres"
//
// URLs: postgres://…, sqlite://path, file:path, :memory:, or a bare path.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"sync/atomic"

	"github.com/lewtec/lewkit/x/cmd"
)

var (
	errNilNew        = errors.New("nil constructor")
	errNotOpen       = errors.New("database not open")
	ErrUnknownScheme = errors.New("unknown database scheme")
)

// DBTX is sqlc's database/sql handle (*sql.DB and *sql.Tx).
type DBTX interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// Arg is a database URL flag. Parse the URL; Value is the connection.
type Arg[Q any] struct {
	raw string
	c   atomic.Pointer[Conn[Q]]
}

// Parse stores the URL.
func (a *Arg[Q]) Parse(s string) error {
	a.raw = s
	a.c.Store(&Conn[Q]{url: s})
	return nil
}

// Value is the connection for this URL. Nil before Parse.
func (a Arg[Q]) Value() *Conn[Q] {
	return a.c.Load()
}

var (
	_ cmd.Parser                  = (*Arg[struct{}])(nil)
	_ cmd.Valuer[*Conn[struct{}]] = Arg[struct{}]{}
)

// Conn migrates and runs queries for one URL.
type Conn[Q any] struct {
	url  string
	conn *sql.DB
	new  func(DBTX) Q
}

// URL is the raw database URL.
func (c *Conn[Q]) URL() string {
	if c == nil {
		return ""
	}
	return c.url
}

// Open connects, runs migrations on fsys, and binds sqlc New.
func (c *Conn[Q]) Open(ctx context.Context, fsys fs.FS, new func(DBTX) Q) error {
	if c == nil {
		return errNotOpen
	}
	if new == nil {
		return errNilNew
	}
	if err := c.Migrate(ctx, fsys); err != nil {
		return err
	}
	c.new = new
	return nil
}

// Migrate connects (if needed) and applies migrations in fsys.
func (c *Conn[Q]) Migrate(ctx context.Context, fsys fs.FS) error {
	if c == nil {
		return errNotOpen
	}
	eng, dsn, err := lookup(c.url)
	if err != nil {
		return err
	}
	if c.conn == nil {
		conn, err := sql.Open(eng.Driver, dsn)
		if err != nil {
			return fmt.Errorf("open %s: %w", eng.Driver, err)
		}
		if err := conn.PingContext(ctx); err != nil {
			return errors.Join(fmt.Errorf("ping %s: %w", eng.Driver, err), conn.Close())
		}
		c.conn = conn
	}
	if fsys == nil || eng.Up == nil {
		return nil
	}
	if err := eng.Up(c.conn, fsys); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// Queries is Q on the connection. Open must have been called.
func (c *Conn[Q]) Queries() Q {
	var z Q
	if c == nil || c.conn == nil || c.new == nil {
		return z
	}
	return c.new(c.conn)
}

// Tx runs fn with Q bound to a transaction.
func (c *Conn[Q]) Tx(ctx context.Context, fn func(Q) error) error {
	if c == nil || c.conn == nil || c.new == nil {
		return errNotOpen
	}
	tx, err := c.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	if err := fn(c.new(tx)); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// Close closes the connection.
func (c *Conn[Q]) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

// Dir is fs.Sub that panics. Use next to //go:embed at package scope.
func Dir(fsys fs.FS, name string) fs.FS {
	sub, err := fs.Sub(fsys, name)
	if err != nil {
		panic(err)
	}
	return sub
}

// As adapts sqlc New (func(DBTX) *Queries) to an interface Q.
func As[Q, C any](new func(DBTX) C) func(DBTX) Q {
	return func(x DBTX) Q {
		return any(new(x)).(Q)
	}
}
