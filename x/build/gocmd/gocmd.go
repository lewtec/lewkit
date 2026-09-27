// Package gocmd runs go build -v and logs each compiler line.
package gocmd

import (
	"context"
	"log/slog"
	"os/exec"
)

// Command is one go invocation. Verb is build or run. Empty Verb is build.
// Each output line is slog.Info.
type Command struct {
	Verb string
	Dir  string
	Env  []string
	Args []string
}

// Run runs go <verb> -v with Args after the verb.
func (c Command) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	verb := c.Verb
	if verb == "" {
		verb = "build"
	}
	argv := append([]string{verb, "-v"}, c.Args...)
	cmd := exec.CommandContext(ctx, "go", argv...)
	cmd.Dir = c.Dir
	cmd.Env = c.Env
	log := slog.NewLogLogger(slog.Default().Handler(), slog.LevelInfo).Writer()
	cmd.Stdout = log
	cmd.Stderr = log
	return cmd.Run()
}
