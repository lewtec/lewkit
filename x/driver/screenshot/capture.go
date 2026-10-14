package screenshot

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/png"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

var (
	ErrCaptureViaCmdDecode        = errors.New("decode")
	ErrCaptureViaCmdFailed        = errors.New("failed")
	ErrExpected4GeometryFieldsGot = errors.New("expected 4 geometry fields, got")
	ErrParseGeometryField         = errors.New("parse geometry field")
)

// CaptureViaCmd runs name with args and decodes stdout as an image.
func CaptureViaCmd(ctx context.Context, name string, args ...string) (image.Image, error) {
	out, err := execdriver.Output(ctx, execdriver.MustCommand(ctx, name, args...))
	if err != nil {
		return nil, fmt.Errorf("%s %w: %w", name, ErrCaptureViaCmdFailed, err)
	}
	img, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		return nil, fmt.Errorf("%w %s output: %w", ErrCaptureViaCmdDecode, name, err)
	}
	return img, nil
}
