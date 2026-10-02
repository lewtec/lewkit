package systemd

import (
	"context"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

type backend struct{}

func (backend) Lock(ctx context.Context) error {
	return execdriver.RunProgram(ctx, "loginctl", "lock-session")
}

func (backend) Logout(ctx context.Context) error {
	return execdriver.RunProgram(ctx, "loginctl", "terminate-session", "self")
}

func (backend) Suspend(ctx context.Context) error {
	return execdriver.RunProgram(ctx, "systemctl", "suspend")
}

func (backend) Hibernate(ctx context.Context) error {
	return execdriver.RunProgram(ctx, "systemctl", "hibernate")
}

func (backend) Reboot(ctx context.Context) error {
	return execdriver.RunProgram(ctx, "systemctl", "reboot")
}

func (backend) Shutdown(ctx context.Context) error {
	return execdriver.RunProgram(ctx, "systemctl", "poweroff")
}
