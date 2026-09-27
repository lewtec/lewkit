package native

import (
	"context"
	"fmt"
	"os/exec"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

type backend struct{}

func (backend) Run(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

func (backend) Which(_ context.Context, name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%w: %s", execdriver.ErrNotFound, name)
	}
	return path, nil
}
