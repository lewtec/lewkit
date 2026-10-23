package gotool

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

// goroot.zip is the go1.27.0 standard library and pkg/include taken from
// that release's binary GOROOT, including src/vendor. The git pin omits
// files that dist generates, including runtime's z*.go and
// internal/buildcfg/zbootstrap.go.
//
//go:embed goroot.zip
var gorootZip []byte

var (
	hashOnce sync.Once
	hashHex  string

	gorootOnce sync.Once
	gorootPath string
	gorootErr  error
)

func gorootHash() string {
	hashOnce.Do(func() {
		sum := sha256.Sum256(gorootZip)
		hashHex = hex.EncodeToString(sum[:8])
	})
	return hashHex
}

func gorootDir() (string, error) {
	gorootOnce.Do(func() {
		gorootPath, gorootErr = extractGoroot()
	})
	return gorootPath, gorootErr
}

func extractGoroot() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	parent := filepath.Join(base, "lewkit", "goroot")
	dir := filepath.Join(parent, "go1.27.0-"+gorootHash())
	marker := filepath.Join(dir, ".complete")
	if _, err := os.Stat(marker); err == nil {
		return dir, nil
	}
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp(parent, "extract-")
	if err != nil {
		return "", err
	}
	ok := false
	defer func() {
		if !ok {
			os.RemoveAll(tmp)
		}
	}()
	if err := unzipGoroot(tmp); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(tmp, ".complete"), nil, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, dir); err != nil {
		if _, statErr := os.Stat(marker); statErr == nil {
			ok = true
			return dir, nil
		}
		return "", err
	}
	ok = true
	return dir, nil
}

func unzipGoroot(dst string) error {
	reader, err := zip.NewReader(bytes.NewReader(gorootZip), int64(len(gorootZip)))
	if err != nil {
		return err
	}
	for _, file := range reader.File {
		name, err := safeZipPath(file.Name)
		if err != nil {
			return err
		}
		if name == "" {
			continue
		}
		target := filepath.Join(dst, filepath.FromSlash(name))
		if file.FileInfo().IsDir() || strings.HasSuffix(file.Name, "/") {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := writeZipFile(file, target); err != nil {
			return err
		}
	}
	return nil
}

func writeZipFile(file *zip.File, target string) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, rc)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func safeZipPath(name string) (string, error) {
	if name == "" {
		return "", nil
	}
	clean := path.Clean("/" + name)
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" || clean == "." {
		return "", nil
	}
	if clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, ":") {
		return "", fmt.Errorf("%w: %s", ErrStdlib, name)
	}
	return clean, nil
}
