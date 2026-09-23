package main

import (
	"context"
	"os"

	"github.com/lewtec/lewkit/x/herdr"
)

type herdrCmd struct {
	reorder *reorderCmd
}

func (herdrCmd) Description() string {
	return "Herdr workspace layout"
}

type reorderCmd struct {
	specs []herdr.RepoBranch `help:"REPO:BRANCH, for example .dotfiles:feat/teste"`
}

func (reorderCmd) Description() string {
	return "nest worktrees, park feature branches, and order workspaces"
}

func (c *reorderCmd) Run(ctx context.Context) error {
	return herdr.Reorder(ctx, herdr.Options{
		Specs: c.specs,
		Out:   os.Stdout,
	})
}
