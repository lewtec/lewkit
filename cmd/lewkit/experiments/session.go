package experiments

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
)

func limitsFrom(ctx context.Context) taskgroup.Limits {
	if l, ok := cmd.Lookup[taskgroup.Limits](ctx, "taskgroup"); ok {
		return l
	}
	return taskgroup.DefaultLimits()
}

func runDemo(ctx context.Context, work func(context.Context) error) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	s, ctx := taskgroup.New(ctx, limitsFrom(ctx))
	return progress.Run(s, progress.WithStop(ctx, stop), work)
}

func runPlain(ctx context.Context, work func(context.Context) error) error {
	s, ctx := taskgroup.New(ctx, limitsFrom(ctx))
	err := work(ctx)
	if werr := s.Wait(); err == nil {
		err = werr
	}
	return err
}
