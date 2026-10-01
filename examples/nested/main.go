// Nested puts child tasks inside an isolated error boundary.
//
//	go run ./cmd/lewkit release run --config ./examples/nested/eletrocromo.json
package main

import (
	"context"
	"time"

	"github.com/lewtec/lewkit/examples/internal/pace"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/taskgroup"
)

func init() { entry.Bind(run) }

func main() { entry.Main(run) }

func run(ctx context.Context) error {
	taskgroup.Go(ctx, "bundle", taskgroup.Control, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("starting bundle phase")
		if err := pace.Sleep(ctx, 40*time.Millisecond); err != nil {
			return err
		}
		err := taskgroup.Isolate(ctx, func(ctx context.Context) error {
			icons := taskgroup.Go(ctx, "bundle:icons", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
				s.Update("generating icons")
				s.Progress(0, 3)
				for i := range 3 {
					s.Progress(int64(i+1), 3)
					if err := pace.Sleep(ctx, 70*time.Millisecond); err != nil {
						return err
					}
				}
				return nil
			})
			taskgroup.Go(ctx, "bundle:manifest", taskgroup.IO, func(ctx context.Context, s *taskgroup.Status) error {
				s.Update("writing manifest.json")
				if err := pace.Sleep(ctx, 90*time.Millisecond); err != nil {
					return err
				}
				return nil
			}, icons)
			return nil
		})
		if err != nil {
			return err
		}
		s.Update("bundle complete")
		return nil
	})
	return nil
}
