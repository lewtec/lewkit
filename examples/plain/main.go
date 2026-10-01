// Plain schedules fetch, process, and write.
//
//	go run ./cmd/lewkit release run --config ./examples/plain/eletrocromo.json
package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lewtec/lewkit/examples/internal/pace"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/taskgroup"
)

func init() { entry.Bind(run) }

func main() { entry.Main(run) }

func run(ctx context.Context) error {
	slog.Info("plain demo")
	fetch := taskgroup.Go(ctx, "fetch", taskgroup.Internet, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("contacting API")
		s.Progress(0, 4)
		for i := 1; i <= 4; i++ {
			s.Progress(int64(i), 4)
			s.Update(fmt.Sprintf("page %d", i))
			if err := pace.Sleep(ctx, 60*time.Millisecond); err != nil {
				return err
			}
		}
		return nil
	})
	proc := taskgroup.Go(ctx, "process", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("crunching numbers")
		if err := pace.Sleep(ctx, 120*time.Millisecond); err != nil {
			return err
		}
		return nil
	}, fetch)
	taskgroup.Go(ctx, "write", taskgroup.IO, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("writing artifacts")
		if err := pace.Sleep(ctx, 80*time.Millisecond); err != nil {
			return err
		}
		return nil
	}, proc)
	return nil
}
