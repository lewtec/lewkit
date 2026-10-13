package swaybg

import (
	"context"
	"fmt"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/driver/wallpaper"
	"github.com/lewtec/lewkit/x/release"
)

type backend struct{}

func (backend) SetStatic(ctx context.Context, path string) error {
	stopWallpaper(ctx)
	swaybg, err := execdriver.Find(ctx, "swaybg")
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

var _ wallpaper.Driver = backend{}
