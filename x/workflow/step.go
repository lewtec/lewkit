package workflow

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/lewtec/lewkit/x/taskgroup"
)

type runtime struct {
	mu      sync.Mutex
	rebuilt map[string]bool
}

func (r *runtime) ran(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rebuilt[name]
}

func (r *runtime) mark(name string) {
	r.mu.Lock()
	r.rebuilt[name] = true
	r.mu.Unlock()
}

func runStep(ctx context.Context, dir string, step Step, status *taskgroup.Status, rt *runtime) error {
	work := step.Dir
	if work == "" {
		work = dir
	}
	if !stale(work, step, rt) {
		status.Update("up to date")
		status.Progress(1, 1)
		return nil
	}
	status.Update(step.title())
	done := status.Unit()
	defer done()
	if len(step.Tasks) == 0 {
		return nil
	}
	if err := mkdirOutputs(work, step.Outputs); err != nil {
		return fmt.Errorf("%s: %w", step.Name, err)
	}
	ctx = withDir(ctx, work)
	for _, task := range step.Tasks {
		if task == nil {
			continue
		}
		if err := task.Run(ctx, status); err != nil {
			return fmt.Errorf("%s: %w", step.Name, err)
		}
	}
	rt.mark(step.Name)
	return nil
}

func stale(dir string, step Step, rt *runtime) bool {
	if step.Phony || len(step.Outputs) == 0 {
		return len(step.Tasks) > 0
	}
	for _, dep := range step.Deps {
		if rt.ran(dep) {
			return true
		}
	}
	var oldest time.Time
	for i, out := range step.Outputs {
		info, err := os.Stat(Join(dir, out))
		if err != nil {
			return true
		}
		if i == 0 || info.ModTime().Before(oldest) {
			oldest = info.ModTime()
		}
	}
	for _, in := range step.Inputs {
		info, err := os.Stat(Join(dir, in))
		if err != nil || info.ModTime().After(oldest) {
			return true
		}
	}
	return false
}
