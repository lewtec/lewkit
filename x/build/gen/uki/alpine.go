package uki

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	alpineRelease = "3.24"
	alpineVersion = "6.18.56-r0"
	alpinePkg     = "linux-virt"
)

// Arch is an Alpine Linux apk architecture.
//
//lewkit:enum
type Arch string

const (
	ArchX86_64  Arch = "x86_64"  // amd64
	ArchAArch64 Arch = "aarch64" // arm64
)

// AlpineArch is the apk architecture for a Go GOARCH.
// amd64 is x86_64. arm64 is aarch64.
func AlpineArch(goarch string) (string, error) {
	switch goarch {
	case "amd64":
		return string(ArchX86_64), nil
	case "arm64":
		return string(ArchAArch64), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrArch, goarch)
	}
}

func kernelURL(goarch string) (string, error) {
	arch, err := AlpineArch(goarch)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://dl-cdn.alpinelinux.org/alpine/v%s/main/%s/%s-%s.apk", alpineRelease, arch, alpinePkg, alpineVersion), nil
}

func cachePath(goarch string) (string, error) {
	arch, err := AlpineArch(goarch)
	if err != nil {
		return "", err
	}
	return filepath.Join(os.TempDir(), "lewkit-alpine", alpineVersion, arch, alpinePkg+".apk"), nil
}

// fetchKernel downloads the pinned Alpine linux-virt apk for goarch.
// A file already in the cache is reused.
func fetchKernel(ctx context.Context, goarch string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	dst, err := cachePath(goarch)
	if err != nil {
		return "", err
	}
	if reusable(dst) {
		return dst, nil
	}
	url, err := kernelURL(goarch)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("uki: %s: %s", url, resp.Status)
	}
	tmp := dst + ".partial"
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		os.Remove(tmp)
		return "", copyErr
	}
	if closeErr != nil {
		os.Remove(tmp)
		return "", closeErr
	}
	if !reusable(tmp) {
		os.Remove(tmp)
		return "", fmt.Errorf("uki: %s: short download", url)
	}
	if err := os.Rename(tmp, dst); err != nil {
		return "", err
	}
	return dst, nil
}

func reusable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.Size() < 8<<20 {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	var magic [2]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		return false
	}
	return magic[0] == 0x1f && magic[1] == 0x8b
}
