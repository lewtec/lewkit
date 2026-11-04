package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/lewtec/lewkit/x/cmd"
)

func main() {
	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

type root struct{}

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
