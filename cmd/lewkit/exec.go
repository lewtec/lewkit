package main

import (
	"context"
	"errors"
	"os"
	"os/exec"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/sops"
)

type execCmd struct {
	env    []sops.File   `short:"e" long:"env" help:"dotenv file. A SOPS age file is decrypted."`
	target cmd.StringArg `short:"t" long:"target" default:"" help:"tool target, such as conda:name"`
	sep    *cmd.Dash
	args   []cmd.StringArg `help:"command and arguments"`
}

func (execCmd) Description() string {
	return "run a command with a decrypted environment"
}

func (c *execCmd) Run(ctx context.Context) error {
	var env sops.Env
	for _, file := range c.env {
		part, err := sops.ParseEnv(file.Value())
		if err != nil {
			return err
		}
		env = env.Merge(part)
	}
	target, err := sops.ParseTarget(c.target.Value())
	if err != nil {
		return err
	}
	err = (sops.Command{
		Env:    env,
		Target: target,
		Args:   cmd.Values(c.args),
	}).Run(ctx)
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}
	return err
}
