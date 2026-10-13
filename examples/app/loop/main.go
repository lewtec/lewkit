// Loop advances five steps on one bar.
//
//	go run ./cmd/lewkit release run --config ./examples/loop/eletrocromo.json
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

func main() { entry.Main(context.Background(), run) }

func run(ctx context.Context) error {
	taskgroup.Go(ctx, "loop-demo", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
		for i := 1; i <= 5; i++ {
			if err := pace.Sleep(ctx, 200*time.Millisecond); err != nil {
				return err
			}
			s.Update(fmt.Sprintf("step %d/5", i))
			s.Progress(int64(i), 5)
		}
		return nil
	})
	return nil
}
