// Map runs one task per path under one bar.
//
//	go run ./cmd/lewkit release run --config ./examples/map/eletrocromo.json
package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/lewtec/lewkit/examples/internal/pace"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/taskgroup"
)

func init() { entry.Bind(run) }

func main() { entry.Main(context.Background(), run) }

func run(ctx context.Context) error {
	items := []string{
		"src/main.go",
		"src/cmd.go",
		"x/taskgroup/map.go",
		"x/taskgroup/tree.go",
		"README.md",
	}
	results, err := taskgroup.Map[string, string]{
		Name:     "demo-map",
		Items:    items,
		PoolKind: taskgroup.IO,
		TaskName: func(_ int, path string) string { return "item:" + path },
		Fn: func(ctx context.Context, st *taskgroup.Status, path string) (string, error) {
			st.Update("starting " + path)
			if err := pace.Sleep(ctx, 80*time.Millisecond+time.Duration(len(path)%4)*30*time.Millisecond); err != nil {
				return "", err
			}
			st.Update("done " + path)
			return "processed:" + path, nil
		},
	}.Run(ctx)
	if err != nil {
		return err
	}
	slog.Info("map finished", "count", len(results))
	return nil
}
