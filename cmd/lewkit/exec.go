package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/sops"
	"github.com/lewtec/lewkit/x/tool"

	_ "github.com/lewtec/lewkit/x/tool/prelude"
)

type execCmd struct {
	env   []sops.File     `short:"e" long:"env" help:"dotenv file. A SOPS age file is decrypted."`
	tools []cmd.StringArg `short:"t" long:"tool" help:"x/tool spec to install before the command. Repeatable."`
	sep   *cmd.Dash
	args  []cmd.StringArg `help:"command and arguments"`
}

func (execCmd) Description() string {
	return "run a command with a decrypted environment and resolved tools"
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
	argv := cmd.Values(c.args)
	if len(argv) == 0 {
		return fmt.Errorf("sops: command is empty")
	}
	command := argv[0]
	specs := cmd.Values(c.tools)
	if len(specs) > 0 {
		path, err := ensureTools(ctx, specs, command)
		if err != nil {
			return err
		}
		command = path
	}
	err := (sops.Command{
		Env:  env,
		Args: append([]string{command}, argv[1:]...),
	}).Run(ctx)
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}
	return err
}

func ensureTools(ctx context.Context, specs []string, command string) (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	store, err := tool.Open(filepath.Join(cache, release.Name(), "tools"))
	if err != nil {
		return "", err
	}
	path, err := store.EnsureCommand(ctx, specs, command)
	if err != nil {
		return "", err
	}
	return path, nil
}
