// Webview serves an in-process page in the system web view.
//
//	go run ./cmd/lewkit release run --config ./examples/webview/eletrocromo.json
package main

import (
	"context"
	"sync/atomic"

	"github.com/lewtec/lewkit/x/app"
	"github.com/lewtec/lewkit/x/entry"
)

func init() { entry.Bind(runApp) }

func main() { entry.Main(context.Background(), runApp) }

func runApp(ctx context.Context) error {
	var count atomic.Int64
	return app.App{
		Title:   "Webview",
		Width:   800,
		Height:  600,
		Handler: app.Web(webviewPage(&count)),
	}.Run(ctx)
}
