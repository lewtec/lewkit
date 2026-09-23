package experiments

import (
	"bytes"
	"context"
	"net/http"
	"testing/fstest"

	"github.com/lewtec/lewkit/x/cmd"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/driver/webview"
	"github.com/lewtec/lewkit/x/http/asset"
	"github.com/lewtec/lewkit/x/http/middleware"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/ui/web"
)

type spaCmd struct {
	width   cmd.IntArg[int] `long:"width" default:"800" help:"window width"`
	height  cmd.IntArg[int] `long:"height" default:"600" help:"window height"`
	profile *cmd.StringArg  `long:"profile" help:"directory for cookies, storage, and cache"`
}

func (spaCmd) Description() string {
	return "templ page served as a single-page app"
}

func (c *spaCmd) Run(ctx context.Context) error {
	return runDemo(ctx, c.run)
}

func (c *spaCmd) run(ctx context.Context) error {
	profile := ""
	if c.profile != nil {
		profile = c.profile.Value()
	}
	view, err := webview.Open(ctx, webview.Config{
		Title:   "lewkit",
		Width:   c.width.Value(),
		Height:  c.height.Value(),
		Profile: profile,
		Handler: spaHandler(ctx),
	})
	if err != nil {
		return err
	}
	taskgroup.Go(ctx, "webview", taskgroup.CPU, func(ctx context.Context, status *taskgroup.Status) error {
		defer view.Close()
		status.Update("open")
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-view.Done():
			return nil
		}
	})
	return nil
}

func spaHandler(ctx context.Context) http.Handler {
	var body bytes.Buffer
	_ = web.Page().Render(ctx, &body)
	files := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: body.Bytes()},
	}
	return asset.Mount(middleware.SPA(files, nil))
}
