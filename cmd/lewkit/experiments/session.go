package experiments

import (
	"context"
	"time"

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
	s, ctx := enter(ctx)
	return progress.Run(s, ctx, work)
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-t.C:
		return nil
	}
}

func runPlain(ctx context.Context, work func(context.Context) error) error {
	s, ctx := enter(ctx)
	err := work(ctx)
	if werr := s.Wait(); err == nil {
		err = werr
	}
	return err
}
