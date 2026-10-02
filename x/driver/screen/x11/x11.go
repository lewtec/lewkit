package x11

import (
	"context"
	"strings"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

type backend struct{}

func (backend) SetDPMS(ctx context.Context, on bool) error {
	state := "off"
	if on {
		state = "on"
	}
	return execdriver.RunProgram(ctx, "xset", "dpms", "force", state)
}

func (backend) IsDPMSOn(ctx context.Context) (bool, error) {
	out, err := execdriver.OutputString(ctx, "xset", "q")
	if err != nil {
		return false, err
	}
	return monitorOn(out), nil
}

func (backend) Reset(ctx context.Context) error {
	return execdriver.RunProgram(ctx, "xrandr", "--auto")
}

func monitorOn(out string) bool {
	return strings.Contains(out, "Monitor is On")
}
