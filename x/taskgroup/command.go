package taskgroup

import (
	"context"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

// RunCommand runs name with args.
// Dir is the working directory. A non-nil env replaces the child environment.
// Stdout and stderr share the session line writer.
func RunCommand(ctx context.Context, name, dir string, env []string, args ...string) error {
	cmd := execdriver.MustCommand(ctx, name, args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = env
	}
	out := LineWriterFrom(ctx)
	cmd.Stdout = out
	cmd.Stderr = out
	return execdriver.Run(ctx, cmd)
}
