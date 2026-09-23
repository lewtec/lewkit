package experiments

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/lewtec/lewkit/x/cmd"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/driver/webview"
	"github.com/lewtec/lewkit/x/taskgroup"
)

type webviewCmd struct {
	width   cmd.IntArg[int] `long:"width" default:"800" help:"window width"`
	height  cmd.IntArg[int] `long:"height" default:"600" help:"window height"`
	profile *cmd.StringArg  `long:"profile" help:"directory for cookies, storage, and cache"`
}

func (webviewCmd) Description() string {
	return "system web view with an in-process handler"
}

func (c *webviewCmd) Run(ctx context.Context) error {
	return runDemo(ctx, c.run)
}

func (c *webviewCmd) run(ctx context.Context) error {
	profile := ""
	if c.profile != nil {
		profile = c.profile.Value()
	}
	var count atomic.Int64
	view, err := webview.Open(ctx, webview.Config{
		Title:   "lewkit webview",
		Width:   c.width.Value(),
		Height:  c.height.Value(),
		Profile: profile,
		Handler: webviewPage(&count),
	})
	if err != nil {
		return err
	}
	taskgroup.Go(ctx, "webview", taskgroup.CPU, func(ctx context.Context, status *taskgroup.Status) error {
		defer view.Close()
		status.Update("0")
		for {
			select {
			case <-ctx.Done():
				return context.Cause(ctx)
			case <-view.Done():
				return nil
			case message := <-view.Messages():
				status.Update(string(message))
			}
		}
	})
	return nil
}

func webviewPage(count *atomic.Int64) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/increment":
			if request.Method != http.MethodPost {
				http.Error(response, "method", http.StatusMethodNotAllowed)
				return
			}
			count.Add(1)
			response.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(response, "%d", count.Load())
		case "/count":
			response.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(response, "%d", count.Load())
		default:
			response.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = response.Write([]byte(webviewHTML))
		}
	})
}

const webviewHTML = `<!doctype html>
<meta charset="utf-8">
<title>lewkit webview</title>
<style>
  body { font: 18px sans-serif; margin: 2rem; background: #111; color: #eee; }
  button { font: inherit; padding: 0.4rem 0.8rem; }
  #count { font-size: 4rem; margin: 0.2rem 0 1rem; }
</style>
<h1>webview</h1>
<p id="count">0</p>
<button id="up" type="button">increment</button>
<script>
async function refresh() {
  const response = await fetch("/count");
  document.querySelector("#count").textContent = await response.text();
}
document.querySelector("#up").addEventListener("click", async () => {
  const response = await fetch("/increment", { method: "POST" });
  const count = await response.text();
  document.querySelector("#count").textContent = count;
  window.lewkit.postMessage(count);
});
refresh();
</script>
`
