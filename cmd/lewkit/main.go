package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/lewtec/lewkit/report/sentry"
	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/db/generate"
	"github.com/lewtec/lewkit/x/generate/prelude"
)

func main() {
	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

type root struct {
	sentry   sentry.Arg `long:"sentry-dsn" env:"SENTRY_DSN" help:"Sentry DSN" default:"https://26fa6b84edbc334b77bf7f6e1d7d69bc@o4508616651505664.ingest.us.sentry.io/4512090764607488"`
	generate *generateCmd
	disasm   *disasmCmd
}

func (r *root) Setup() error {
	return r.sentry.Setup()
}

type generateCmd struct {
	db      *dbCmd
	prelude *preludeCmd
}

func (generateCmd) Description() string {
	return "generate code"
}

type dbCmd struct {
	dir cmd.WorkDirArg `help:"directory with sqlite/ and postgres/"`
}

func (dbCmd) Description() string {
	return "generate sqlc packages and a shared Queries interface"
}

func (c *dbCmd) Run(ctx context.Context) error {
	return generate.Run(ctx, c.dir.Value())
}

type preludeCmd struct {
	dir cmd.WorkDirArg `help:"directory to scan for root.go"`
	out *cmd.StringArg `help:"prelude.go to write; stdout if omitted"`
}

func (preludeCmd) Description() string {
	return "blank-import prelude from root.go files"
}

func (c *preludeCmd) Run(ctx context.Context) error {
	dest := ""
	if c.out != nil {
		dest = c.out.Value()
	}
	return prelude.Run(ctx, c.dir.Value(), dest)
}

func (root) Description() string {
	return "Well planned primitives to be used in other projects."
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	app, err := cmd.Parse[cmd.App[root]](os.Args[1:]...)
	if err != nil {
		return err
	}
	return app.Run(ctx)
}
