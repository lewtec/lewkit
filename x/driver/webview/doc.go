// Package webview opens an OS web view and wires the page to Go in-process.
//
// There is no loopback listener and no browser binary. The window is the
// system web view: WebKitGTK 6 on Linux, WKWebView on macOS, and WebView2
// on Windows. The page is memory or an fs.FS. JavaScript calls
// window.lewkit.postMessage. Go calls View.Evaluate.
// Config.Profile is the directory for that view's cookies and storage.
// Config.Handler answers origin requests in-process. Nothing listens.
//
// On macOS, call thread.Bind from main and run thread.Loop so AppKit
// receives events.
//
// Blank-import the driver so Open can find it:
//
//	import _ "github.com/lewtec/lewkit/x/driver/prelude"
//
//	view, err := webview.Open(ctx, webview.Config{
//		Title: "hi",
//		HTML:  "<p>hello</p>",
//	})
package webview
