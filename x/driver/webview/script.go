package webview

import (
	"fmt"

	"github.com/lewtec/lewkit/x/release"
)

// ScriptName is the page object and the script-message handler.
// It follows [release.Name], so a name override moves the bridge with it.
func ScriptName() string { return release.Name() }

// WebKitBridge is the injected page object for a WebKit message handler.
func WebKitBridge() string {
	n := ScriptName()
	return fmt.Sprintf(`window.%s={postMessage:function(value){window.webkit.messageHandlers.%s.postMessage(value);}};`, n, n)
}

// ChromeBridge is the injected page object for WebView2.
func ChromeBridge() string {
	n := ScriptName()
	return fmt.Sprintf(`window.%s={postMessage:function(value){window.chrome.webview.postMessage(value);}};`, n)
}

// HostSuffix is the fake host on the WebView2 asset origin.
func HostSuffix() string { return "." + ScriptName() + ".invalid" }
