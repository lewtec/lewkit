package cmd

import (
	"fmt"

	"github.com/lewtec/lewkit/x/path"
)

// DataDirArg is a path that must exist and be a directory.
type DataDirArg struct {
	Container[string]
}

func (d *DataDirArg) Parse(arg string) error {
	return parseExisting(&d.Container, arg)
}

func parseExisting(c *Container[string], arg string) error {
	if err := existingDir(arg); err != nil {
		return err
	}
	c.value = arg
	return nil
}

func existingDir(arg string) error {
	root, err := path.Open(arg)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidArgument, err)
	}
	return root.Close()
}

var (
	_ Parser      = (*DataDirArg)(nil)
	_ Arg[string] = (*DataDirArg)(nil)
)
