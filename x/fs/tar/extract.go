package tar

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ExtractFirst writes the first member of a gzip tar into dir.
// The destination name is the member's base name and the mode is 0o755.
func ExtractFirst(archive, dir string) (string, error) {
	file, err := os.Open(archive)
	if err != nil {
		return "", err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	header, err := tr.Next()
	if err == io.EOF {
		return "", fmt.Errorf("empty archive %s", archive)
	}
	if err != nil {
		return "", err
	}
	return writeMember(dir, header.Name, tr)
}

func writeMember(dir, name string, r io.Reader) (string, error) {
	dest := filepath.Join(dir, filepath.Base(name))
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, r); err != nil {
		out.Close()
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	return dest, nil
}
