// Tasks shows progress bars, logs, pools, and dependencies.
//
//	go run ./cmd/lewkit release run --config ./examples/tasks/eletrocromo.json
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/lewtec/lewkit/examples/internal/pace"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/taskgroup"
)

func init() { entry.Bind(run) }

func main() { entry.Main(context.Background(), run) }

var errSimulated503 = errors.New("simulated 503 from registry")

func run(ctx context.Context) error {
	slog.Info("scheduling demo tasks")

	dl := taskgroup.Go(ctx, "bundle.tar.gz", taskgroup.Internet, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("connecting")
		if err := pace.Sleep(ctx, 80*time.Millisecond); err != nil {
			return err
		}
		const total int64 = 10 * 1024 * 1024
		s.Progress(0, total)
		for i := 1; i <= 10; i++ {
			cur := total * int64(i) / 10
			s.Progress(cur, total)
			s.Update(fmt.Sprintf("%.1f MiB / %.0f MiB", float64(cur)/(1024*1024), float64(total)/(1024*1024)))
			if err := pace.Sleep(ctx, 50*time.Millisecond); err != nil {
				return err
			}
		}
		return nil
	})

	build := taskgroup.Go(ctx, "build", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
		for step := 1; step <= 4; step++ {
			s.Update(fmt.Sprintf("part %d/4", step))
			s.Progress(int64(step), 4)
			if err := pace.Sleep(ctx, 90*time.Millisecond); err != nil {
				return err
			}
		}
		return nil
	}, dl)

	taskgroup.Go(ctx, "check", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("static analysis")
		if err := pace.Sleep(ctx, 200*time.Millisecond); err != nil {
			return err
		}
		return nil
	}, dl)

	taskgroup.Go(ctx, "install", taskgroup.IO, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("installing")
		done := s.Unit()
		if err := pace.Sleep(ctx, 120*time.Millisecond); err != nil {
			return err
		}
		done()
		return nil
	}, build)

	taskgroup.Go(ctx, "lint", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("linting workspace")
		if err := pace.Sleep(ctx, 240*time.Millisecond); err != nil {
			return err
		}
		return nil
	})

	taskgroup.Go(ctx, "publish", taskgroup.Internet, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("connecting to registry")
		if err := pace.Sleep(ctx, 160*time.Millisecond); err != nil {
			return err
		}
		return errSimulated503
	}, build)
	return nil
}
