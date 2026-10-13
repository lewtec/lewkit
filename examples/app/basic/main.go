// Basic is a one-line web handler.
//
//	go run ./cmd/lewkit release run --config ./examples/basic/eletrocromo.json
package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/lewtec/lewkit/x/app"
	"github.com/lewtec/lewkit/x/entry"
)

func init() { entry.Bind(runApp) }

func main() { entry.Main(context.Background(), runApp) }

func runApp(ctx context.Context) error {
	return app.App{
		Title:   "Basic",
		Handler: app.Web(newPage()),
	}.Run(ctx)
}

func newPage() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprintf(w, "it works!"); err != nil {
			return
		}
	})
}
