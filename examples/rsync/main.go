// Rsync rewrites one row per fake transfer.
//
//	go run ./cmd/lewkit release run --config ./examples/rsync/eletrocromo.json
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
	const step = 80 * time.Millisecond
	files := []struct {
		name string
		size int64
	}{
		{"bundle.tar.gz", 80 << 20},
		{"icons.zip", 12 << 20},
		{"docs.pdf", 3 << 20},
		{"video.mp4", 140 << 20},
	}
	for _, f := range files {
		taskgroup.Go(ctx, f.name, taskgroup.IO, func(ctx context.Context, s *taskgroup.Status) error {
			w := taskgroup.LineWriterFrom(ctx)
			defer w.Close()
			s.Progress(0, f.size)
			const ticks = 20
			start := time.Now()
			for i := 1; i <= ticks; i++ {
				cur := f.size * int64(i) / ticks
				s.Progress(cur, f.size)
				elapsed := time.Since(start).Seconds()
				if elapsed < 0.001 {
					elapsed = 0.001
				}
				speed := float64(cur) / (1024 * 1024) / elapsed
				pct := int(cur * 100 / f.size)
				if _, err := fmt.Fprintf(w, "%-16s %3d%%  %6.1fMB/s\r", f.name, pct, speed); err != nil {
					return err
				}
				if err := pace.Sleep(ctx, step); err != nil {
					return err
				}
			}
			s.Update("done")
			_, err := fmt.Fprintf(w, "%-16s done\n", f.name)
			return err
		})
	}
	return nil
}
