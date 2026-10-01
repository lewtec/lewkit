package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

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
  body { font: 18px sans-serif; margin: 2rem; background: #fff; color: #111; }
  @media (prefers-color-scheme: dark) {
    body { background: #111; color: #eee; }
  }
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
