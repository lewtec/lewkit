package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Up applies iofs migrations at "." on an open connection.
// name is the golang-migrate database instance name (postgres, sqlite3, …).
func Up(ctx context.Context, conn *sql.DB, fsys fs.FS, name string, instance func(*sql.DB) (database.Driver, error)) error {
	if err := ctx.Err(); err != nil {
		return context.Cause(ctx)
	}
	src, err := iofs.New(fsys, ".")
	if err != nil {
		return fmt.Errorf("iofs: %w", err)
	}
	inst, err := instance(conn)
	if err != nil {
		return fmt.Errorf("instance: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, name, inst)
	if err != nil {
		return err
	}
	stop := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			m.GracefulStop <- true
		case <-stop:
		}
	}()
	err = m.Up()
	close(stop)
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}
		return err
	}
	return nil
}
