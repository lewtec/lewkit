package main

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
	"github.com/lewtec/lewkit/x/workflow"
	makefile "github.com/lewtec/lewkit/x/workflow/make"
	"github.com/lewtec/lewkit/x/workflow/ninja"
)

type workflowCmd struct {
	make  *workflowMakeCmd
	ninja *workflowNinjaCmd
}

func (workflowCmd) Description() string {
	return "run a build graph from a makefile or a ninja file"
}

type workflowMakeCmd struct {
	file    cmd.StringArg   `short:"f" long:"file" default:"Makefile" help:"makefile to read"`
	dir     cmd.WorkDirArg  `short:"C" long:"directory" help:"directory recipes run in"`
	noPrint cmd.Flag        `long:"no-print-directory" help:"accepted and ignored"`
	rest    []cmd.StringArg `help:"targets, or NAME=value"`
}

func (workflowMakeCmd) Description() string {
	return "build targets from a makefile"
}

func (c *workflowMakeCmd) Run(ctx context.Context) error {
	assigns, targets := makefile.ParseArgs(cmd.Values(c.rest))
	file := workflow.Join(c.dir.Value(), c.file.Value())
	dir := c.dir.Value()
	return onSession(ctx, "make", targets, func(ctx context.Context, st *taskgroup.Status) (workflow.Graph, error) {
		st.Update(file)
		parsed, err := makefile.Load(ctx, file, dir, assigns, targets)
		if err != nil {
			return workflow.Graph{}, err
		}
		return parsed.Graph(ctx, targets)
	})
}

// onSession reads the file as a task so the progress view opens during load.
// Recipe steps are children of that task. The task returns after scheduling
// them; waiting inside it would hold a pool worker until they finished.
// entry.Run already shows this session, so progress.Run joins that view and
// does not start another. With no session yet, the root taskgroup limits
// apply and this call waits.
func onSession(ctx context.Context, name string, targets []string, load func(context.Context, *taskgroup.Status) (workflow.Graph, error)) error {
	session, ctx := sessionFrom(ctx)
	return progress.Run(session, ctx, func(ctx context.Context) error {
		taskgroup.Go(ctx, name, taskgroup.CPU, func(ctx context.Context, st *taskgroup.Status) error {
			graph, err := load(ctx, st)
			if err != nil {
				return err
			}
			return workflow.Run(ctx, graph, targets)
		})
		return nil
	})
}

type workflowNinjaCmd struct {
	file cmd.StringArg   `short:"f" long:"file" default:"build.ninja" help:"ninja file to read"`
	dir  cmd.WorkDirArg  `short:"C" long:"directory" help:"directory commands run in"`
	rest []cmd.StringArg `help:"targets"`
}

func (workflowNinjaCmd) Description() string {
	return "build targets from a ninja file"
}

func (c *workflowNinjaCmd) Run(ctx context.Context) error {
	targets := cmd.Values(c.rest)
	file := workflow.Join(c.dir.Value(), c.file.Value())
	dir := c.dir.Value()
	return onSession(ctx, "ninja", targets, func(ctx context.Context, st *taskgroup.Status) (workflow.Graph, error) {
		st.Update(file)
		parsed, err := ninja.Load(ctx, file, dir)
		if err != nil {
			return workflow.Graph{}, err
		}
		return parsed.Graph(targets)
	})
}
