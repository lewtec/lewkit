package experiments

import (
	"context"
	"time"

	"github.com/lewtec/lewkit/x/taskgroup"
)

type nestedCmd struct{}

func (nestedCmd) Description() string {
	return "Isolate as an error boundary with child tasks"
}

func (*nestedCmd) Run(ctx context.Context) error {
	return runDemo(ctx, func(ctx context.Context) error {
		taskgroup.Go(ctx, "bundle", taskgroup.Control, func(ctx context.Context, s *taskgroup.Status) error {
			s.Update("starting bundle phase")
			time.Sleep(40 * time.Millisecond)
			err := taskgroup.Isolate(ctx, func(ctx context.Context) error {
				icons := taskgroup.Go(ctx, "bundle:icons", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
					s.Update("generating icons")
					s.Progress(0, 3)
					for i := range 3 {
						s.Progress(int64(i+1), 3)
						time.Sleep(70 * time.Millisecond)
					}
					return nil
				})
				taskgroup.Go(ctx, "bundle:manifest", taskgroup.IO, func(ctx context.Context, s *taskgroup.Status) error {
					s.Update("writing manifest.json")
					time.Sleep(90 * time.Millisecond)
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
	})
}
