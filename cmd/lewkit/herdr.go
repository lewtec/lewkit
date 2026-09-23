package main

import (
	"context"
	"os"

	"github.com/lewtec/lewkit/x/herdr"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
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
	session, ctx := taskgroup.New(ctx, taskgroup.DefaultLimits())
	var report herdr.Report
	err := progress.Run(session, ctx, func(ctx context.Context) error {
		taskgroup.Go(ctx, "reorder", taskgroup.Control, func(ctx context.Context, status *taskgroup.Status) error {
			var runErr error
			report, runErr = herdr.Reorder(ctx, herdr.Options{
				Specs:  c.specs,
				Status: status,
			})
			return runErr
		})
		return nil
	})
	if err != nil {
		return err
	}
	_, err = os.Stdout.WriteString(progress.Format(orderNodes(report), 0))
	return err
}

func orderNodes(report herdr.Report) []taskgroup.Node {
	nodes := make([]taskgroup.Node, len(report.Order))
	for i, space := range report.Order {
		state := taskgroup.Done
		if space.RepoRoot == "" {
			state = taskgroup.Failed
		}
		nodes[i] = taskgroup.Node{
			ID:      taskgroup.ID(i + 1),
			Name:    space.Label,
			Message: herdr.Kind(space) + " " + herdr.Explain(report.Pin, space),
			State:   state,
			Pool:    taskgroup.IO,
		}
	}
	return nodes
}
