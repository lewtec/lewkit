package x11

import (
	"context"
	"fmt"
	"strings"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

type backend struct{}

func (backend) SetDPMS(ctx context.Context, on bool) error {
	state := "off"
	if on {
		state = "on"
	}
	return run(ctx, "xset", "dpms", "force", state)
}

func (backend) IsDPMSOn(ctx context.Context) (bool, error) {
	out, err := output(ctx, "xset", "q")
	if err != nil {
		return false, err
	}
	return monitorOn(out), nil
}

func (backend) Reset(ctx context.Context) error {
	return run(ctx, "xrandr", "--auto")
}

func monitorOn(out string) bool {
	return strings.Contains(out, "Monitor is On")
}

func run(ctx context.Context, name string, args ...string) error {
	cmd := execdriver.MustCommand(ctx, name, args...)
	if err := execdriver.Run(ctx, cmd); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func output(ctx context.Context, name string, args ...string) (string, error) {
	cmd := execdriver.MustCommand(ctx, name, args...)
	out, err := execdriver.Output(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return string(out), nil
}
