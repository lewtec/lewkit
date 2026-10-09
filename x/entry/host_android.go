//go:build android

package entry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sync/atomic"

	"github.com/lewtec/lewkit/x/driver/thread"
	"github.com/lewtec/lewkit/x/ffi/jni"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

type surfaceBox struct {
	ptr    uintptr
	width  int
	height int
}

var surfaceCh = make(chan surfaceBox, 1)

var errNoSurface = errors.New("android surface closed")

// ShowSurface tells the host the Go app is a native window, not a page.
func ShowSurface() {
	if _, err := jni.CallStatic("lewkit.Host", "openSurface"); err != nil {
		slog.Error("android surface", "err", err)
	}
}

// RequestSurface asks the Android host for a native window and waits for it.
// The size is the surface the activity reported, which is the GUI window size.
func RequestSurface(ctx context.Context) (uintptr, int, int, error) {
	if _, err := jni.CallStatic("lewkit.Host", "openSurface"); err != nil {
		return 0, 0, 0, err
	}
	select {
	case box := <-surfaceCh:
		if box.ptr == 0 || box.width < 1 || box.height < 1 {
			return 0, 0, 0, errNoSurface
		}
		return box.ptr, box.width, box.height, nil
	case <-ctx.Done():
		return 0, 0, 0, ctx.Err()
	}
}

// AndroidHost reports whether JNI_OnLoad noted a Java VM in this process.
// The activity did. The packaged fallback executes this library as a
// process, notes nothing, and has to publish the loopback ready line.
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

var surfaceFn atomic.Value

// HandleSurface receives a replacement Android native window and its size.
func HandleSurface(fn func(ptr uintptr, width, height int)) {
	if fn == nil {
		return
	}
	surfaceFn.Store(fn)
}

// publishSurface queues the window and tells a registered handler.
// Android stores a tag in the top byte of some pointers; the window
// pointer the GUI uses is the low 56 bits.
func publishSurface(ptr uintptr, width, height int) {
	if ptr != 0 {
		ptr &= 0x00ffffffffffffff
	}
	box := surfaceBox{ptr: ptr, width: width, height: height}
	select {
	case surfaceCh <- box:
	default:
		select {
		case <-surfaceCh:
		default:
		}
		surfaceCh <- box
	}
	if fn, ok := surfaceFn.Load().(func(uintptr, int, int)); ok && fn != nil {
		fn(ptr, width, height)
	}
}

// hookCall is lewkit.Hook.call. name selects the host event.
// args is a jobjectArray. The return is a jlong; void events return 0.
func hookCall(env, namePtr, argsPtr uintptr) (ret uintptr) {
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error("jni hook", "panic", recovered)
			ret = 0
		}
	}()
	name, err := jni.UTF(env, namePtr)
	if err != nil || name == "" {
		slog.Error("jni hook", "err", err)
		return 0
	}
	if name == "start" {
		runtime.LockOSThread()
		jni.SetCurrentEnv(func() uintptr { return jni.Attach() })
		jni.SetRunner(func(fn func()) { thread.Do(fn) })
		if err := jni.Bind(env, "lewkit/Host"); err != nil {
			slog.Error("jni bind", "err", err)
			return 0
		}
	}
	args, err := jni.Objects(env, argsPtr)
	if err != nil {
		slog.Error("jni hook", "name", name, "err", err)
		return 0
	}
	defer releaseHookArgs(args)
	switch name {
	case "start":
		runHost(hookString(args, 0))
	case "window":
		return hookWindow(env, args)
	case "pointer":
		DeliverPointer(hookInt(args, 0), hookInt(args, 1), hookInt(args, 2))
	case "resize":
		DeliverResize(hookInt(args, 0), hookInt(args, 1))
	case "lost":
		DeliverSurfaceLost()
	case "obscure":
		DeliverInsets(
			hookInt(args, 0),
			hookInt(args, 1),
			hookInt(args, 2),
			hookInt(args, 3),
			hookInt(args, 4),
			hookInt(args, 5),
		)
	default:
		slog.Error("jni hook", "name", name)
	}
	return 0
}

func hookWindow(env uintptr, args []any) uintptr {
	ptr, err := androidffi.WindowFromSurface(env, hookPeer(args, 0))
	if err != nil {
		slog.Error("android window", "err", err)
		ptr = 0
	}
	publishSurface(ptr, hookInt(args, 1), hookInt(args, 2))
	return ptr & 0x00ffffffffffffff
}

func hookString(args []any, i int) string {
	if i >= len(args) {
		return ""
	}
	s, _ := args[i].(string)
	return s
}

func hookInt(args []any, i int) int {
	if i >= len(args) || args[i] == nil {
		return 0
	}
	switch n := args[i].(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	default:
		return 0
	}
}

func hookPeer(args []any, i int) uintptr {
	if i >= len(args) {
		return 0
	}
	ref, _ := args[i].(*jni.Ref)
	return ref.Peer()
}

func releaseHookArgs(args []any) {
	for _, arg := range args {
		if ref, ok := arg.(*jni.Ref); ok {
			ref.Release()
		}
	}
}

// runHost is Hook.call("start") after the JNIEnv is bound.
// The library skips main, so the app registered with Bind runs here.
// ELETROCROMO_NO_UI asks for a loopback URL. This process has a Java VM,
// so the window stays the activity page. The fallback process has no VM.
func runHost(path string) {
	_ = os.Setenv("ELETROCROMO_NO_UI", "1")
	_ = os.Setenv("ELETROCROMO_READY_FILE", path)
	root := context.Background()
	thread.Bind(root)
	defer func() {
		if recovered := recover(); recovered != nil {
			NotifyFail(fmt.Sprint(recovered))
		}
	}()
	if err := RunBound(root); err != nil {
		NotifyFail(err.Error())
	}
}
