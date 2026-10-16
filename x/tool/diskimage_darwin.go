//go:build darwin

package tool

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

var (
	errDitto = errors.New("ditto")
)

func mountDiskImage(ctx context.Context, source string) (string, func(), error) {
	mountpoint, err := os.MkdirTemp("", "lewkit-dmg-")
	if err != nil {
		return "", nil, err
	}
	output, err := runCombined(ctx, "hdiutil", "attach", "-nobrowse", "-readonly", "-noautoopen", "-quiet", "-mountpoint", mountpoint, source)
	if err != nil {
		_, _ = runCombined(context.WithoutCancel(ctx), "hdiutil", "detach", "-quiet", mountpoint)
		os.RemoveAll(mountpoint)
		return "", nil, fmt.Errorf("%w: hdiutil attach: %w: %s", ErrDiskImageMount, err, commandOutput(output))
	}
	cleanup := func() {
		_, _ = runCombined(context.WithoutCancel(ctx), "hdiutil", "detach", "-quiet", mountpoint)
		_ = os.RemoveAll(mountpoint)
	}
	return mountpoint, cleanup, nil
}

func copyVolume(ctx context.Context, source, destination string) error {
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		from := filepath.Join(source, entry.Name())
		to := filepath.Join(destination, entry.Name())
		// ditto follows a symlink given as the source path. A volume's
		// Applications link must stay a link, not a copy of /Applications.
		info, err := os.Lstat(from)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(from)
			if err != nil {
				return err
			}
			if err := os.Symlink(target, to); err != nil {
				return err
			}
			continue
		}
		output, err := runCombined(ctx, "ditto", from, to)
		if err != nil {
			return fmt.Errorf("%w %s: %w: %s", errDitto, entry.Name(), err, commandOutput(output))
		}
	}
	return nil
}

func runCombined(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := execdriver.MustCommand(ctx, name, args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := execdriver.Run(ctx, cmd)
	return output.Bytes(), err
}

func commandOutput(output []byte) string {
	text := strings.TrimSpace(string(output))
	if len(text) > 512 {
		text = text[:512]
	}
	return text
}
