package io

import (
	"errors"
	"os"
	"path/filepath"
)

// Mkdirp creates dir and any missing parents.
func Mkdirp(dir string) error {
	err := os.Mkdir(dir, 0o755)
	if err == nil || errors.Is(err, os.ErrExist) {
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	parent := filepath.Dir(dir)
	if parent == dir {
		return err
	}
	if err := Mkdirp(parent); err != nil {
		return err
	}
	err = os.Mkdir(dir, 0o755)
	if err == nil || errors.Is(err, os.ErrExist) {
		return nil
	}
	return err
}
