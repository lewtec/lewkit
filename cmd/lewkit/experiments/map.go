package experiments

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lewtec/lewkit/x/taskgroup"
)

type mapCmd struct{}

func (mapCmd) Description() string {
	return "Map over a list under one aggregate bar"
}

func (*mapCmd) Run(ctx context.Context) error {
	return runDemo(ctx, func(ctx context.Context) error {
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
				time.Sleep(80*time.Millisecond + time.Duration(len(path)%4)*30*time.Millisecond)
				st.Update("done " + path)
				return "processed:" + path, nil
			},
		}.Run(ctx)
		if err != nil {
			return err
		}
		slog.Info("map finished", "count", len(results))
		return nil
	})
}

type manyCmd struct{}

func (manyCmd) Description() string {
	return "256 Map items; the view only walks List(n) rows"
}

func (*manyCmd) Run(ctx context.Context) error {
	return runDemo(ctx, func(ctx context.Context) error {
		items := make([]int, 256)
		for i := range items {
			items[i] = i
		}
		_, err := taskgroup.Map[int, int]{
			Name:     "many",
			Items:    items,
			PoolKind: taskgroup.CPU,
			TaskName: func(_ int, n int) string { return fmt.Sprintf("cpu:%d", n) },
			Fn: func(ctx context.Context, st *taskgroup.Status, n int) (int, error) {
				st.Update(fmt.Sprintf("item %d", n))
				time.Sleep(20 * time.Millisecond)
				return n, nil
			},
		}.Run(ctx)
		return err
	})
}
