// Package webview2 opens the Windows WebView2 runtime without cgo.
//
// The runtime is the Edge WebView2 already installed with Windows or Edge.
// This package loads EmbeddedBrowserWebView.dll from that install.
// WEBVIEW2_LOADER, when set, is a full path to WebView2Loader.dll and is used
// instead. A missing runtime is ErrUnavailable. This package does not listen
// and does not ship a loader.
package webview2
