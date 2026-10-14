package tool

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/errs"
	lewfs "github.com/lewtec/lewkit/x/fs"
	lewpath "github.com/lewtec/lewkit/x/path"
)

// ErrDiskImageMount is returned when a disk image cannot be mounted.
var ErrDiskImageMount = errs.Wrap(errs.ErrUnavailable, "mount disk image")

// extractDiskImage mounts source and copies the volume into destination.
// One leading directory is removed when it is not itself an app bundle,
// so Blender/Blender.app lands as Blender.app.
func extractDiskImage(ctx context.Context, source, destination string) error {
	mountpoint, cleanup, err := mountDiskImage(ctx, source)
	if err != nil {
		return err
	}
	defer cleanup()
	return copyDiskImage(ctx, mountpoint, destination)
}

// copyDiskImage copies a mounted volume into destination.
func copyDiskImage(ctx context.Context, mountpoint, destination string) error {
	source, err := diskImageVolumeRoot(mountpoint)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	return copyVolume(ctx, source, destination)
}

// diskImageVolumeRoot returns the directory whose children are the install tree.
// A volume that is only a wrapper folder is unwrapped. An .app at the volume root stays put.
func diskImageVolumeRoot(mountpoint string) (string, error) {
	entries, err := os.ReadDir(mountpoint)
	if err != nil {
		return "", err
	}
	if len(entries) == 1 && entries[0].IsDir() && !isAppBundleName(entries[0].Name()) {
		return filepath.Join(mountpoint, entries[0].Name()), nil
	}
	return mountpoint, nil
}

func isAppBundleName(name string) bool {
	return len(name) > len(".app") && strings.EqualFold(name[len(name)-len(".app"):], ".app")
}

func appBundleStem(name string) string {
	if isAppBundleName(name) {
		return name[:len(name)-len(".app")]
	}
	return name
}

// copyVolumeWalk writes source's children into destination, including symlinks.
func copyVolumeWalk(ctx context.Context, source, destination string) error {
	dest, err := lewpath.Open(destination)
	if err != nil {
		return err
	}
	defer dest.Close()
	return lewfs.Copy(ctx, dest, lewfs.Walk(ctx, os.DirFS(source), nil))
}

// isDiskImageFile reports a .dmg path or a UDIF image (koly trailer).
func isDiskImageFile(hostPath string) bool {
	if strings.HasSuffix(strings.ToLower(hostPath), ".dmg") {
		return true
	}
	file, err := os.Open(hostPath)
	if err != nil {
		return false
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 512 {
		return false
	}
	magic := make([]byte, 4)
	if _, err := file.ReadAt(magic, info.Size()-512); err != nil {
		return false
	}
	return string(magic) == "koly"
}
