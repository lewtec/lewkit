package sway

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type backend struct{}

func (backend) SetDPMS(ctx context.Context, on bool) error {
	state := "off"
	if on {
		state = "on"
	}
	return run(ctx, "swaymsg", "output * dpms "+state)
}

func (backend) IsDPMSOn(ctx context.Context) (bool, error) {
	out, err := output(ctx, "swaymsg", "-t", "get_outputs")
	if err != nil {
		return false, err
	}
	return dpmsOn(out), nil
}

func (backend) Reset(ctx context.Context) error {
	return run(ctx, "swaymsg", "output", "*", "enable")
}

func dpmsOn(out string) bool {
	return strings.Contains(out, `"dpms": true`)
}

func run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func output(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return string(out), nil
}
