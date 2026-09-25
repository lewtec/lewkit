package clipboard

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"io"
	"os/exec"

	"github.com/lewtec/lewkit/x/driver"
)

// PipeText writes text to name's stdin.
func PipeText(ctx context.Context, text, name string, args ...string) error {
	return pipe(ctx, name, args, func(w io.Writer) error {
		_, err := io.WriteString(w, text)
		return err
	})
}

// PipeImage PNG-encodes img into name's stdin.
func PipeImage(ctx context.Context, img image.Image, name string, args ...string) error {
	return pipe(ctx, name, args, func(w io.Writer) error {
		return png.Encode(w, img)
	})
}

func pipe(ctx context.Context, name string, args []string, write func(io.Writer) error) error {
	cmd := exec.CommandContext(ctx, name, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	werr := write(stdin)
	cerr := stdin.Close()
	if err := cmd.Wait(); err != nil {
		if werr != nil {
			return werr
		}
		return fmt.Errorf("%s: %w", name, err)
	}
	if werr != nil {
		return werr
	}
	return cerr
}

// RequireTool reports ErrIncompatible when name is not on PATH.
func RequireTool(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%w: %s not found", driver.ErrIncompatible, name)
	}
	return nil
}

// RequireEnvTool reports ErrIncompatible when envKey is empty or name is missing.
func RequireEnvTool(ctx context.Context, envKey, name string) error {
	if err := driver.RequireEnv(ctx, envKey); err != nil {
		return err
	}
	return RequireTool(name)
}
