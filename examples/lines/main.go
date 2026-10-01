// Lines rewrites three rows until a newline.
//
//	go run ./cmd/lewkit release run --config ./examples/lines/eletrocromo.json
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lewtec/lewkit/examples/internal/pace"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/taskgroup"
)

func init() { entry.Bind(run) }

func main() { entry.Main(run) }

func run(ctx context.Context) error {
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
				if err := pace.Sleep(ctx, job.tick); err != nil {
					return err
				}
			}
			_, err := fmt.Fprintf(w, "%s done\n", job.name)
			return err
		})
	}
	return nil
}
