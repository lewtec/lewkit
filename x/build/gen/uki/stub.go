package uki

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Debian systemd-boot-efi 257.13. The linux*.efi.stub is the UKI stub.
// Newer stubs hang on some firmware, so this stays on the 257 line.
const stubVersion = "257.13-1~deb13u1"

var stubSHA = map[string]string{
	"amd64": "9c623743834bb904a3cb941092cffc958c31230263ce896d332a75e9ec48aa85",
	"arm64": "2ca0d2b4e21d10dd67acf564e58570b6d36359e6d9e9aba43d50fc57996dcf94",
}

func stubName(goarch string) (string, error) {
	switch goarch {
	case "amd64":
		return "linuxx64.efi.stub", nil
	case "arm64":
		return "linuxaa64.efi.stub", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrArch, goarch)
	}
}

func stubURL(goarch string) (string, error) {
	switch goarch {
	case "amd64", "arm64":
	default:
		return "", fmt.Errorf("%w: %s", ErrArch, goarch)
	}
	return fmt.Sprintf("https://deb.debian.org/debian/pool/main/s/systemd/systemd-boot-efi_%s_%s.deb", stubVersion, goarch), nil
}

func fetchStub(ctx context.Context, goarch string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	want, ok := stubSHA[goarch]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrArch, goarch)
	}
	name, err := stubName(goarch)
	if err != nil {
		return nil, err
	}
	dst := filepath.Join(os.TempDir(), "lewkit-uki", stubVersion, goarch, "stub.efi")
	if raw, err := os.ReadFile(dst); err == nil && sha256Hex(raw) == want {
		return raw, nil
	}
	url, err := stubURL(goarch)
	if err != nil {
		return nil, err
	}
	deb, err := httpGet(ctx, url)
	if err != nil {
		return nil, err
	}
	raw, err := debMember(ctx, deb, name)
	if err != nil {
		return nil, err
	}
	if got := sha256Hex(raw); got != want {
		return nil, fmt.Errorf("uki: stub %s sha256 %s", goarch, got)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(dst, raw, 0o644); err != nil {
		return nil, err
	}
	return raw, nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func httpGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("uki: %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// debMember returns one file from a .deb data tarball. name is the base name.
func debMember(ctx context.Context, deb []byte, name string) ([]byte, error) {
	data, err := arMember(deb, "data.tar.xz")
	if err != nil {
		return nil, err
	}
	raw, err := unxz(ctx, data)
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(bytes.NewReader(raw))
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("uki: deb has no %s", name)
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(hdr.Name) != name {
			continue
		}
		return io.ReadAll(tr)
	}
}

func arMember(deb []byte, name string) ([]byte, error) {
	if !bytes.HasPrefix(deb, []byte("!<arch>\n")) {
		return nil, errors.New("uki: deb is not an ar archive")
	}
	off := 8
	for off+60 <= len(deb) {
		hdr := deb[off : off+60]
		member := strings.TrimRight(string(hdr[:16]), " /")
		size, err := strconv.Atoi(strings.TrimSpace(string(hdr[48:58])))
		if err != nil {
			return nil, fmt.Errorf("uki: deb member size: %w", err)
		}
		start := off + 60
		end := start + size
		if end > len(deb) {
			return nil, errors.New("uki: deb member overflows")
		}
		if member == name {
			return deb[start:end], nil
		}
		off = end
		if off%2 == 1 {
			off++
		}
	}
	return nil, fmt.Errorf("uki: deb has no %s", name)
}

func unxz(ctx context.Context, data []byte) ([]byte, error) {
	xz, err := exec.LookPath("xz")
	if err != nil {
		return nil, errors.New("uki: xz is required to unpack the systemd stub")
	}
	cmd := exec.CommandContext(ctx, xz, "-dc")
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("uki: xz: %w", err)
	}
	return out, nil
}
