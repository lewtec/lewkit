package swaybg

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/driver/wallpaper"
)

type backend struct{}

func (backend) SetStatic(ctx context.Context, path string) error {
	stopWallpaper(ctx)
	swaybg, err := look(ctx, "swaybg")
	if err != nil {
		return err
	}
	cmd := execdriver.MustRun(ctx, "systemd-run", "--user", "-u", "lewkit-wallpaper", "--collect", swaybg, "-i", path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("can't run swaybg in systemd unit: %w", err)
	}
	return nil
}

func stopWallpaper(ctx context.Context) {
	_ = execdriver.MustRun(ctx, "systemctl", "--user", "stop", "lewkit-wallpaper.service").Run()
}

func look(ctx context.Context, name string) (string, error) {
	path, err := execdriver.Which(ctx, name)
	if err != nil {
		return "", fmt.Errorf("%w: %s not found", driver.ErrIncompatible, name)
	}
	return path, nil
}

func requireBinary(ctx context.Context, name string) error {
	_, err := look(ctx, name)
	return err
}

var _ wallpaper.Driver = backend{}
