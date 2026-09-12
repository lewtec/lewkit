package cmd

import (
	"errors"
	"fmt"
	"os"
)

var (
	errEmptyPath = errors.New("empty path")
	errNotDir    = errors.New("not a directory")
)

// DataDirArg is a path that must exist and be a directory.
type DataDirArg struct {
	Container[string]
}

func (d *DataDirArg) Parse(arg string) error {
	if arg == "" {
		return fmt.Errorf("%w: %w", ErrInvalidArgument, errEmptyPath)
	}
	st, err := os.Stat(arg)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidArgument, err)
	}
	if !st.IsDir() {
		return fmt.Errorf("%w: %w: %s", ErrInvalidArgument, errNotDir, arg)
	}
	d.value = arg
	return nil
}

var (
	_ Parser      = (*DataDirArg)(nil)
	_ Arg[string] = (*DataDirArg)(nil)
)
