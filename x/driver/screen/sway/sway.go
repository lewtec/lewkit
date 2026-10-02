package sway

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
	return execdriver.RunProgram(ctx, "swaymsg", "output * dpms "+state)
}

func (backend) IsDPMSOn(ctx context.Context) (bool, error) {
	out, err := execdriver.OutputString(ctx, "swaymsg", "-t", "get_outputs")
	if err != nil {
		return false, err
	}
	return dpmsOn(out), nil
}

func (backend) Reset(ctx context.Context) error {
	return execdriver.RunProgram(ctx, "swaymsg", "output", "*", "enable")
}

func dpmsOn(out string) bool {
	return strings.Contains(out, `"dpms": true`)
}
