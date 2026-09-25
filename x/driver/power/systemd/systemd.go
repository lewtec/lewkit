package systemd

import (
	"context"
	"fmt"
	"os/exec"
)

type backend struct{}

func (backend) Lock(ctx context.Context) error {
	return run(ctx, "loginctl", "lock-session")
}

func (backend) Logout(ctx context.Context) error {
	return run(ctx, "loginctl", "terminate-session", "self")
}

func (backend) Suspend(ctx context.Context) error {
	return run(ctx, "systemctl", "suspend")
}

func (backend) Hibernate(ctx context.Context) error {
	return run(ctx, "systemctl", "hibernate")
}

func (backend) Reboot(ctx context.Context) error {
	return run(ctx, "systemctl", "reboot")
}

func (backend) Shutdown(ctx context.Context) error {
	return run(ctx, "systemctl", "poweroff")
}

func run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}
