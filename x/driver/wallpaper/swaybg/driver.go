package swaybg

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/driver/wallpaper"
	"github.com/lewtec/lewkit/x/release"
)

type backend struct{}

func (backend) SetStatic(ctx context.Context, path string) error {
	stopWallpaper(ctx)
	swaybg, err := look(ctx, "swaybg")
	if err != nil {
		return err
	}
	unit := release.Name() + "-wallpaper"
	cmd := execdriver.MustCommand(ctx, "systemd-run", "--user", "-u", unit, "--collect", swaybg, "-i", path)
	if err := execdriver.Run(ctx, cmd); err != nil {
		return fmt.Errorf("can't run swaybg in systemd unit: %w", err)
	}
	return nil
}

func stopWallpaper(ctx context.Context) {
	unit := release.Name() + "-wallpaper.service"
	_ = execdriver.Run(ctx, execdriver.MustCommand(ctx, "systemctl", "--user", "stop", unit))
}

func look(ctx context.Context, name string) (string, error) {
	path, err := execdriver.Which(ctx, name)
	if err != nil {
		return "", fmt.Errorf("%w: %s not found", driver.ErrIncompatible, name)
	}
	return path, nil
}

var _ wallpaper.Driver = backend{}
