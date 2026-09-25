package feh

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/wallpaper"
)

type backend struct{}

func (backend) SetStatic(ctx context.Context, path string) error {
	feh, err := look("feh")
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, feh, "--bg-fill", path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("feh: %w", err)
	}
	return nil
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
