package taskgroup

import "context"

// Isolate runs fn under an unnamed error-boundary node. Failures cancel
// only that subtree; parent siblings keep running. The boundary itself
// does not appear in List.
func Isolate(ctx context.Context, fn func(context.Context) error) error {
	if fn == nil {
		return nil
	}
	parent := FromContext(ctx)
	if parent == nil {
		return fn(ctx)
	}
	id := parent.goTask(ctx, "", Control, func(ctx context.Context, _ *Status) error {
		err := fn(ctx)
		if werr := parent.waitLive(ctx, taskFromContext(ctx)); werr != nil && err == nil {
			err = werr
		}
		return err
	}, true)
	return parent.waitTask(ctx, id)
}

// GoIsolated schedules one named task on an isolated subtree and returns
// its error without cancelling sibling work on the parent session.
func GoIsolated(ctx context.Context, name string, pool PoolKind, fn func(context.Context, *Status) error) error {
	if fn == nil {
		return nil
	}
	if FromContext(ctx) == nil {
		return fn(ctx, &Status{})
	}
	return Isolate(ctx, func(ctx context.Context) error {
		var taskErr error
		Go(ctx, name, pool, func(ctx context.Context, s *Status) error {
			taskErr = fn(ctx, s)
			return taskErr
		})
		sess := MustFromContext(ctx)
		if werr := sess.waitLive(ctx, taskFromContext(ctx)); werr != nil && taskErr == nil {
			return werr
		}
		return taskErr
	})
}
