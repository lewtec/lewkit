package gui

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"

	lewpath "github.com/lewtec/lewkit/x/path"
)

const recentLimit = 8

// Directory is one folder on the welcome list.
type Directory struct {
	Path string
}

// Recent reads the newest folders first. A missing file is an empty list.
func Recent() ([]Directory, error) {
	file, err := recentFile()
	if err != nil {
		return nil, err
	}
	handle, err := os.Open(file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer handle.Close()
	var out []Directory
	seen := map[string]bool{}
	scanner := bufio.NewScanner(handle)
	for scanner.Scan() {
		dir, ok := cleanDir(scanner.Text())
		if !ok || seen[dir] {
			continue
		}
		seen[dir] = true
		out = append(out, Directory{Path: dir})
		if len(out) == recentLimit {
			break
		}
	}
	return out, scanner.Err()
}

// Remember puts dir first in the recent list.
func Remember(dir string) error {
	cleaned, ok := cleanDir(dir)
	if !ok {
		return os.ErrNotExist
	}
	file, err := recentFile()
	if err != nil {
		return err
	}
	previous, err := Recent()
	if err != nil {
		return err
	}
	lines := make([]string, 0, recentLimit)
	lines = append(lines, cleaned)
	for _, item := range previous {
		if item.Path == cleaned || len(lines) == recentLimit {
			continue
		}
		lines = append(lines, item.Path)
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return err
	}
	return os.WriteFile(file, []byte(strings.Join(lines, "\n")+"\n"), 0o600)
}

func recentFile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "lewkit", "recent-dirs"), nil
}

func cleanDir(dir string) (string, bool) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", false
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", false
	}
	root, err := lewpath.Open(abs)
	if err != nil {
		return "", false
	}
	root.Close()
	return abs, true
}
