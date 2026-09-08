package io

import (
	"errors"
	"os"
)

func Mkdirp(dir string) error {
	err := os.Mkdir(dir, 0o755)
	if errors.Is(err, os.ErrExist) {
		return nil
	}
	return err
}
