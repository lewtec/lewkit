package experiments

import (
	"context"
	"fmt"
	"time"

	"github.com/lewtec/lewkit/x/taskgroup"
)

type loopCmd struct{}

func (loopCmd) Description() string {
	return "five steps with a moving bar"
}

func (*loopCmd) Run(ctx context.Context) error {
	return runDemo(ctx, func(ctx context.Context) error {
		taskgroup.Go(ctx, "loop-demo", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
			for i := 1; i <= 5; i++ {
				time.Sleep(200 * time.Millisecond)
				s.Update(fmt.Sprintf("step %d/5", i))
				s.Progress(int64(i), 5)
			}
			return nil
		})
		return nil
	})
}
