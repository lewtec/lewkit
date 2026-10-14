//go:build !darwin

package tool

import (
	"context"
	"fmt"
)

func mountDiskImage(context.Context, string) (string, func(), error) {
	return "", nil, fmt.Errorf("%w: hdiutil", ErrDiskImageMount)
}

func copyVolume(ctx context.Context, source, destination string) error {
	return copyVolumeWalk(ctx, source, destination)
}
