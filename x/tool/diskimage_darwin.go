//go:build darwin

package tool

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func mountDiskImage(ctx context.Context, source string) (string, func(), error) {
	mountpoint, err := os.MkdirTemp("", "lewkit-dmg-")
	if err != nil {
		return "", nil, err
	}
	command := exec.CommandContext(ctx, "hdiutil", "attach", "-nobrowse", "-readonly", "-noautoopen", "-quiet", "-mountpoint", mountpoint, source)
	output, err := command.CombinedOutput()
	if err != nil {
		_ = exec.Command("hdiutil", "detach", "-quiet", mountpoint).Run()
		os.RemoveAll(mountpoint)
		return "", nil, fmt.Errorf("%w: hdiutil attach: %w: %s", ErrDiskImageMount, err, commandOutput(output))
	}
	cleanup := func() {
		_ = exec.Command("hdiutil", "detach", "-quiet", mountpoint).Run()
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
		command := exec.CommandContext(ctx, "ditto", from, to)
		output, err := command.CombinedOutput()
		if err != nil {
			return fmt.Errorf("ditto %s: %w: %s", entry.Name(), err, commandOutput(output))
		}
	}
	return nil
}

func commandOutput(output []byte) string {
	text := strings.TrimSpace(string(output))
	if len(text) > 512 {
		text = text[:512]
	}
	return text
}
