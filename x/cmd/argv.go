package cmd

import (
	"os"
	"path/filepath"
)

// Name is the base name of this process.
func Name() string {
	if len(os.Args) == 0 {
		return ""
	}
	return filepath.Base(os.Args[0])
}

// Args is the process arguments after the program name.
func Args() []string {
	if len(os.Args) < 2 {
		return nil
	}
	return os.Args[1:]
}
