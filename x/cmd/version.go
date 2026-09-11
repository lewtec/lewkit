package cmd

import (
	"context"
	"os"

	"github.com/lewtec/lewkit/x/release"
)

// VersionCmd prints the build version. Opt in with a *VersionCmd field.
type VersionCmd struct{}

func (VersionCmd) Description() string {
	return "print version"
}

func (VersionCmd) Run(context.Context) error {
	return release.PrintVersion(os.Stdout)
}
