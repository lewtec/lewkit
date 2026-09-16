package experiments

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
)

func enter(ctx context.Context) (*taskgroup.Session, context.Context) {
	if s := taskgroup.FromContext(ctx); s != nil {
		return s, ctx
	}
	if arg, ok := cmd.Lookup[taskgroup.Arg](ctx, "taskgroup"); ok {
		return arg.Enter(ctx, taskgroup.DefaultLimits())
	}
	return taskgroup.New(ctx, taskgroup.DefaultLimits())
}

func runDemo(ctx context.Context, work func(context.Context) error) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	s, ctx := enter(ctx)
	return progress.Run(s, progress.WithStop(ctx, stop), work)
}

func runPlain(ctx context.Context, work func(context.Context) error) error {
	s, ctx := enter(ctx)
	err := work(ctx)
	if werr := s.Wait(); err == nil {
		err = werr
	}
	return err
}
