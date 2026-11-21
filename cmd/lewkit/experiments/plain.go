package experiments

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lewtec/lewkit/x/taskgroup"
)

type plainCmd struct{}

func (plainCmd) Description() string {
	return "same scheduling without the progress TUI"
}

func (*plainCmd) Run(ctx context.Context) error {
	return runPlain(ctx, func(ctx context.Context) error {
		slog.Info("plain demo: no TUI")
		fetch := taskgroup.Go(ctx, "fetch", taskgroup.Internet, func(ctx context.Context, s *taskgroup.Status) error {
			s.Update("contacting API")
			s.Progress(0, 4)
			for i := 1; i <= 4; i++ {
				s.Progress(int64(i), 4)
				s.Update(fmt.Sprintf("page %d", i))
				if err := sleep(ctx, 60*time.Millisecond); err != nil {
					return err
				}
			}
			return nil
		})
		proc := taskgroup.Go(ctx, "process", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
			s.Update("crunching numbers")
			if err := sleep(ctx, 120*time.Millisecond); err != nil {
				return err
			}
			return nil
		}, fetch)
		taskgroup.Go(ctx, "write", taskgroup.IO, func(ctx context.Context, s *taskgroup.Status) error {
			s.Update("writing artifacts")
			if err := sleep(ctx, 80*time.Millisecond); err != nil {
				return err
			}
			return nil
		}, proc)
		return nil
	})
}
