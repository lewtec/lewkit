// Package gocmd runs go build or go run.
package gocmd

import (
	"context"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/taskgroup"
)

// Command is one go invocation. Verb is build or run. Empty Verb is build.
// The child process writes to the session line writer.
type Command struct {
	Verb string
	Dir  string
	Env  []string
	Args []string
}

// Run runs go <verb> -v with Args after the verb.
// With a taskgroup session on ctx, the command is an IO subtask.
func (c Command) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	verb := c.Verb
	if verb == "" {
		verb = "build"
	}
	if taskgroup.FromContext(ctx) == nil {
		return c.exec(ctx, verb)
	}
	errCh := make(chan error, 1)
	taskgroup.Go(ctx, "go "+verb, taskgroup.IO, func(ctx context.Context, st *taskgroup.Status) error {
		st.Update(verb)
		err := c.exec(ctx, verb)
		errCh <- err
		return err
	})
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c Command) exec(ctx context.Context, verb string) error {
	return run(ctx, "go", c.Dir, c.Env, append([]string{verb, "-v"}, c.Args...)...)
}

// Tool runs name as one IO task through the exec driver.
// Stdout and stderr share that task's progress row.
func Tool(ctx context.Context, name, dir string, env []string, args ...string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if taskgroup.FromContext(ctx) == nil {
		return run(ctx, name, dir, env, args...)
	}
	errCh := make(chan error, 1)
	taskgroup.Go(ctx, name, taskgroup.IO, func(ctx context.Context, st *taskgroup.Status) error {
		st.Update(name)
		err := run(ctx, name, dir, env, args...)
		errCh <- err
		return err
	})
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func run(ctx context.Context, name, dir string, env []string, args ...string) error {
	cmd := execdriver.MustCommand(name, args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = env
	}
	out := taskgroup.LineWriterFrom(ctx)
	cmd.Stdout = out
	cmd.Stderr = out
	return execdriver.Run(ctx, cmd)
}
