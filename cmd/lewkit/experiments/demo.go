package experiments

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/lewtec/lewkit/x/taskgroup"
)

var errSimulated503 = errors.New("simulated 503 from registry")

// Demo is `lewkit experiments demo`.
type Demo struct {
	Tasks  *tasksCmd
	Plain  *plainCmd
	Nested *nestedCmd
	Loop   *loopCmd
	Map    *mapCmd `cmd:"map"`
	Many   *manyCmd
}

func (Demo) Description() string {
	return "showcase the taskgroup executor and progress view"
}

type tasksCmd struct{}

func (tasksCmd) Description() string {
	return "progress bars, logs, pools, and dependencies"
}

func (*tasksCmd) Run(ctx context.Context) error {
	return runDemo(ctx, scheduleTasks)
}

func scheduleTasks(ctx context.Context) error {
	slog.Info("scheduling demo tasks")

	dl := taskgroup.Go(ctx, "bundle.tar.gz", taskgroup.Internet, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("connecting")
		time.Sleep(80 * time.Millisecond)
		const total int64 = 10 * 1024 * 1024
		s.Progress(0, total)
		for i := 1; i <= 10; i++ {
			cur := total * int64(i) / 10
			s.Progress(cur, total)
			s.Update(fmt.Sprintf("%.1f MiB / %.0f MiB", float64(cur)/(1024*1024), float64(total)/(1024*1024)))
			time.Sleep(50 * time.Millisecond)
		}
		return nil
	})

	build := taskgroup.Go(ctx, "build", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
		for step := 1; step <= 4; step++ {
			s.Update(fmt.Sprintf("part %d/4", step))
			s.Progress(int64(step), 4)
			time.Sleep(90 * time.Millisecond)
		}
		return nil
	}, dl)

	taskgroup.Go(ctx, "check", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("static analysis")
		time.Sleep(200 * time.Millisecond)
		return nil
	}, dl)

	taskgroup.Go(ctx, "install", taskgroup.IO, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("installing")
		done := s.Unit()
		time.Sleep(120 * time.Millisecond)
		done()
		return nil
	}, build)

	taskgroup.Go(ctx, "lint", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("linting workspace")
		time.Sleep(240 * time.Millisecond)
		return nil
	})

	taskgroup.Go(ctx, "publish", taskgroup.Internet, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("connecting to registry")
		time.Sleep(160 * time.Millisecond)
		return errSimulated503
	}, build)
	return nil
}
