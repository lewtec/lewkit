package experiments

import (
	"context"
	"fmt"
	"time"

	"github.com/lewtec/lewkit/x/taskgroup"
)

type linesCmd struct{}

func (linesCmd) Description() string {
	return "three LineWriter rows that rewrite with CR until a newline"
}

func (*linesCmd) Run(ctx context.Context) error {
	return runDemo(ctx, func(ctx context.Context) error {
		jobs := []struct {
			name string
			n    int
			tick time.Duration
		}{
			{"alpha", 24, 80 * time.Millisecond},
			{"beta", 16, 120 * time.Millisecond},
			{"gamma", 20, 100 * time.Millisecond},
		}
		for _, job := range jobs {
			taskgroup.Go(ctx, job.name, taskgroup.Control, func(ctx context.Context, s *taskgroup.Status) error {
				w := taskgroup.LineWriterFrom(ctx)
				defer w.Close()
				s.Update("streaming")
				for i := 1; i <= job.n; i++ {
					if _, err := fmt.Fprintf(w, "%s %d/%d\r", job.name, i, job.n); err != nil {
						return err
					}
					if err := sleep(ctx, job.tick); err != nil {
						return err
					}
				}
				_, err := fmt.Fprintf(w, "%s done\n", job.name)
				return err
			})
		}
		return nil
	})
}
