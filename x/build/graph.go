package build

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/workflow"
)

// runGraph runs g on the workflow engine.
// A session already on ctx is not waited on by workflow.Run. This waits
// until every step has run, or ctx ends. The caller is a Control task:
// waiting here does not hold a pool worker the steps need.
func runGraph(ctx context.Context, g workflow.Graph) error {
	if taskgroup.FromContext(ctx) == nil || len(g.Steps) == 0 {
		return workflow.Run(ctx, g, nil)
	}
	done := make(chan struct{})
	var left atomic.Int32
	left.Store(int32(len(g.Steps)))
	var mu sync.Mutex
	var stepErr error
	fail := func(err error) {
		if err == nil {
			return
		}
		mu.Lock()
		if stepErr == nil {
			stepErr = err
		}
		mu.Unlock()
	}
	for i := range g.Steps {
		tasks := g.Steps[i].Tasks
		g.Steps[i].Tasks = []workflow.Task{
			workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
				defer func() {
					if left.Add(-1) == 0 {
						close(done)
					}
				}()
				for _, task := range tasks {
					if task == nil {
						continue
					}
					if err := task.Run(ctx, st); err != nil {
						fail(err)
						return err
					}
				}
				return nil
			}),
		}
	}
	if err := workflow.Run(ctx, g, nil); err != nil {
		return err
	}
	select {
	case <-done:
		mu.Lock()
		err := stepErr
		mu.Unlock()
		return err
	case <-ctx.Done():
		mu.Lock()
		err := stepErr
		mu.Unlock()
		if err != nil {
			return err
		}
		return context.Cause(ctx)
	}
}
