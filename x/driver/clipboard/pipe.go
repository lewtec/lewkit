package clipboard

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

var (
	ErrPipe = errors.New("")
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
	cmd := execdriver.MustCommand(ctx, name, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := execdriver.Start(ctx, cmd); err != nil {
		return fmt.Errorf("%w%s: %w", ErrPipe, name, err)
	}
	werr := write(stdin)
	cerr := stdin.Close()
	if err := execdriver.Wait(ctx, cmd); err != nil {
		if werr != nil {
			return werr
		}
		return fmt.Errorf("%w%s: %w", ErrPipe, name, err)
	}
	if werr != nil {
		return werr
	}
	return cerr
}

// RequireTool reports ErrIncompatible when name is not on PATH.
func RequireTool(ctx context.Context, name string) error {
	return execdriver.RequireBinary(ctx, name)
}

// RequireEnvTool reports ErrIncompatible when envKey is empty or name is missing.
func RequireEnvTool(ctx context.Context, envKey, name string) error {
	return execdriver.RequireEnvBinary(ctx, envKey, name)
}
