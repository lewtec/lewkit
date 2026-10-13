package feh

import (
	"context"
	"fmt"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/driver/wallpaper"
)

type backend struct{}

func (backend) SetStatic(ctx context.Context, path string) error {
	feh, err := execdriver.Find(ctx, "feh")
	if err != nil {
		return err
	}
	cmd := execdriver.MustCommand(ctx, feh, "--bg-fill", path)
	if err := execdriver.Run(ctx, cmd); err != nil {
		return fmt.Errorf("feh: %w", err)
	}
	return nil
}

var _ wallpaper.Driver = backend{}
