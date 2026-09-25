package swaybg

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/wallpaper"
)

type backend struct{}

func (backend) SetStatic(ctx context.Context, path string) error {
	stopWallpaper(ctx)
	swaybg, err := look("swaybg")
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "systemd-run", "--user", "-u", "lewkit-wallpaper", "--collect", swaybg, "-i", path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("can't run swaybg in systemd unit: %w", err)
	}
	return nil
}

func stopWallpaper(ctx context.Context) {
	_ = exec.CommandContext(ctx, "systemctl", "--user", "stop", "lewkit-wallpaper.service").Run()
}

func look(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%w: %s not found", driver.ErrIncompatible, name)
	}
	return path, nil
}

func requireBinary(name string) error {
	_, err := look(name)
	return err
}

var _ wallpaper.Driver = backend{}
