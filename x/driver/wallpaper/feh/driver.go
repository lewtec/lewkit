package feh

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/driver/wallpaper"
)

type backend struct{}

func (backend) SetStatic(ctx context.Context, path string) error {
	feh, err := look(ctx, "feh")
	if err != nil {
		return err
	}
	cmd := execdriver.MustCommand(ctx, feh, "--bg-fill", path)
	if err := execdriver.Run(ctx, cmd); err != nil {
		return fmt.Errorf("feh: %w", err)
	}
	return nil
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
