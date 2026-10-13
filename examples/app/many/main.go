// Many maps 256 items. The view calls List(n).
//
//	go run ./cmd/lewkit release run --config ./examples/many/eletrocromo.json
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
			if err := pace.Sleep(ctx, 20*time.Millisecond); err != nil {
				return 0, err
			}
			return n, nil
		},
	}.Run(ctx)
	return err
}
