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
	Tree   *treeCmd
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
		if err := sleep(ctx, 80*time.Millisecond); err != nil {
			return err
		}
		const total int64 = 10 * 1024 * 1024
		s.Progress(0, total)
		for i := 1; i <= 10; i++ {
			cur := total * int64(i) / 10
			s.Progress(cur, total)
			s.Update(fmt.Sprintf("%.1f MiB / %.0f MiB", float64(cur)/(1024*1024), float64(total)/(1024*1024)))
			if err := sleep(ctx, 50*time.Millisecond); err != nil {
				return err
			}
		}
		return nil
	})

	build := taskgroup.Go(ctx, "build", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
		for step := 1; step <= 4; step++ {
			s.Update(fmt.Sprintf("part %d/4", step))
			s.Progress(int64(step), 4)
			if err := sleep(ctx, 90*time.Millisecond); err != nil {
				return err
			}
		}
		return nil
	}, dl)

	taskgroup.Go(ctx, "check", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("static analysis")
		if err := sleep(ctx, 200*time.Millisecond); err != nil {
			return err
		}
		return nil
	}, dl)

	taskgroup.Go(ctx, "install", taskgroup.IO, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("installing")
		done := s.Unit()
		if err := sleep(ctx, 120*time.Millisecond); err != nil {
			return err
		}
		done()
		return nil
	}, build)

	taskgroup.Go(ctx, "lint", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("linting workspace")
		if err := sleep(ctx, 240*time.Millisecond); err != nil {
			return err
		}
		return nil
	})

	taskgroup.Go(ctx, "publish", taskgroup.Internet, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("connecting to registry")
		if err := sleep(ctx, 160*time.Millisecond); err != nil {
			return err
		}
		return errSimulated503
	}, build)
	return nil
}
