package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/db/generate"
)

func main() {
	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

type root struct {
	generate *generateCmd
}

type generateCmd struct {
	db *dbCmd
}

func (generateCmd) Description() string {
	return "generate code"
}

func (c *generateCmd) Run(ctx context.Context) error {
	text, err := cmd.Usage[generateCmd]("lewkit generate")
	if err != nil {
		return err
	}
	_, err = os.Stdout.WriteString(text)
	return err
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
