package zip

import (
	stdzip "archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ExtractFirst writes the first member of a zip into dir.
// The destination name is the member's base name and the mode is 0o755.
func ExtractFirst(archive, dir string) (string, error) {
	zr, err := stdzip.OpenReader(archive)
	if err != nil {
		return "", err
	}
	defer zr.Close()
	if len(zr.File) == 0 {
		return "", fmt.Errorf("empty archive %s", archive)
	}
	entry := zr.File[0]
	in, err := entry.Open()
	if err != nil {
		return "", err
	}
	defer in.Close()
	dest := filepath.Join(dir, filepath.Base(entry.Name))
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return "", err
	}
	return dest, nil
}
