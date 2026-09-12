package cmd

import (
	"errors"
	"fmt"
	"os"
)

var (
	ErrEmptyPath = errors.New("empty path")
	ErrNotDir    = errors.New("not a directory")
)

// DataDirArg is a path that must exist and be a directory.
type DataDirArg struct {
	Container[string]
}

func (d *DataDirArg) Parse(arg string) error {
	if arg == "" {
		return fmt.Errorf("%w: %w", ErrInvalidArgument, ErrEmptyPath)
	}
	st, err := os.Stat(arg)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidArgument, err)
	}
	if !st.IsDir() {
		return fmt.Errorf("%w: %w: %s", ErrInvalidArgument, ErrNotDir, arg)
	}
	d.value = arg
	return nil
}

var (
	_ Parser      = (*DataDirArg)(nil)
	_ Arg[string] = (*DataDirArg)(nil)
)
