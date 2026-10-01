// Files asks for files and prints each path.
//
//	go run ./cmd/lewkit release run --config ./examples/filedialog/eletrocromo.json
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lewtec/lewkit/x/driver/filedialog"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/entry"
)

func init() { entry.Bind(run) }

func main() { entry.Main(run) }

func run(ctx context.Context) error {
	paths, err := filedialog.Choose(ctx, filedialog.Request{Title: "Choose files"})
	if err != nil {
		return err
	}
	for _, path := range paths {
		fmt.Println(path)
	}
	slog.Info("filedialog", "paths", len(paths))
	return nil
}
