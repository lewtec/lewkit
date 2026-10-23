// Package gocmd runs go build or go run.
// Compiler host runs the go command. Compiler lewkit builds a CGO-free
// amd64 or arm64 binary when x/build/gocmd/lewkit is linked. Release
// commands pass that choice through the context. Their default is lewkit.
package gocmd

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/taskgroup"
)

// Command is one go invocation. Verb is build or run. Empty Verb is build.
// The child process writes to the session line writer.
// Compiler selects the host go command or the embedded compiler.
// The zero value reads the compiler stored by [WithCompiler].
// That is also host when the context has none.
type Command struct {
	Verb     string
	Dir      string
	Env      []string
	Args     []string
	Compiler Compiler
}

// Run runs go <verb> -v with Args after the verb.
// With a taskgroup session on ctx, the host go command is an IO subtask.
// The lewkit compiler is a Control task named "go build". Each package
// compile and the link are CPU children of that task.
func (c Command) Run(ctx context.Context) error {
	if c.active(ctx) == CompilerLewkit {
		if runLewkit == nil {
			return fmt.Errorf("%w: embedded compiler is not linked", ErrCompiler)
		}
		return runLewkit(ctx, c)
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
		return context.Cause(ctx)
	}
}

func (c Command) exec(ctx context.Context, verb string) error {
	return taskgroup.RunCommand(ctx, "go", c.Dir, c.Env, append([]string{verb, "-v"}, c.Args...)...)
}

// Tool runs name as one IO task through the exec driver.
// Stdout and stderr share that task's progress row.
func Tool(ctx context.Context, name, dir string, env []string, args ...string) error {
	if taskgroup.FromContext(ctx) == nil {
		return taskgroup.RunCommand(ctx, name, dir, env, args...)
	}
	errCh := make(chan error, 1)
	taskgroup.Go(ctx, name, taskgroup.IO, func(ctx context.Context, st *taskgroup.Status) error {
		st.Update(name)
		err := taskgroup.RunCommand(ctx, name, dir, env, args...)
		errCh <- err
		return err
	})
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}
