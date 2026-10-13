//go:build android

package entry

import (
	"log/slog"

	_ "github.com/lewtec/lewkit/x/entry/android"
	"github.com/lewtec/lewkit/x/entry/internal/androidcall"
	"github.com/lewtec/lewkit/x/ffi/jni"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

func init() {
	androidcall.PointerFunc = DeliverPointer
	androidcall.ResizeFunc = DeliverResize
	androidcall.LostFunc = DeliverSurfaceLost
	androidcall.FontFunc = DeliverFontScale
	androidcall.InsetsFunc = DeliverInsets
	androidcall.RunFunc = RunBound
	androidcall.FailFunc = NotifyFail
}

// ShowSurface tells the host the Go app is a native window, not a page.
func ShowSurface() {
	if _, err := jni.CallStatic("lewkit.Host", "openSurface"); err != nil {
		slog.Error("android surface", "err", err)
	}
}

// AndroidHost reports whether JNI_OnLoad noted a Java VM in this process.
// The activity did. A process that did not is not that activity, so a web
// handler publishes the loopback ready line.
func AndroidHost() bool {
	return androidffi.NotedVM()
}

// NotifyReady tells the Android host that the web window can open this URL.
func NotifyReady(url string) {
	if _, err := jni.CallStatic("lewkit.Host", "ready", url); err != nil {
		slog.Error("android ready", "err", err)
	}
}

// NotifyFail tells the Android host why startup stopped.
func NotifyFail(message string) {
	if _, err := jni.CallStatic("lewkit.Host", "fail", message); err != nil {
		slog.Error("android fail", "err", err)
	}
}
