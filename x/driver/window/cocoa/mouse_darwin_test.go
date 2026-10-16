//go:build darwin

package cocoa

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver/thread"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ffi/native"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if os.Getenv("LEWKIT_CLICK_PROBE") != "1" {
		os.Exit(m.Run())
	}
	os.Exit(probeContentClick())
}

func TestContentViewStubsMouseTracking(t *testing.T) {
	_, err := native.Open("/System/Library/Frameworks/Cocoa.framework/Cocoa", native.Global|native.Lazy)
	require.NoError(t, err)
	require.NoError(t, ensureDropClass())
	view := objc.ID(dropClass).Send(selAllocView)
	view = view.Send(selInitWithFrame, nsRect{})
	require.NotEqual(t, objc.ID(0), view)
	t.Cleanup(func() { view.Send(selRelease) })
	for _, name := range []string{
		"mouseDown:",
		"mouseUp:",
		"rightMouseDown:",
		"rightMouseUp:",
		"otherMouseDown:",
		"otherMouseUp:",
		"mouseDragged:",
		"rightMouseDragged:",
		"otherMouseDragged:",
		"acceptsFirstMouse:",
	} {
		require.Truef(t, responds(view, objc.RegisterName(name)), "content view %s", name)
	}
}

// probeContentClick opens a window on the process main thread and sends a
// left mouse down through AppKit. NSView's own mouseDown waits until mouse
// up, so a return means the content view no longer tracks.
func probeContentClick() int {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := thread.Run(ctx, func(ctx context.Context) error {
		host, err := window.Open(ctx, window.Config{Title: "click", Width: 240, Height: 160})
		if err != nil {
			return err
		}
		defer host.Close()
		surf, ok := host.(window.Surfacer)
		if !ok || surf.Surface().A == 0 {
			return fmt.Errorf("window has no view")
		}
		view := objc.ID(surf.Surface().A)
		elapsed := make(chan time.Duration, 1)
		go func() {
			var took time.Duration
			thread.Do(func() {
				start := time.Now()
				if err := sendMouseDown(view); err != nil {
					fmt.Fprintln(os.Stderr, err)
					took = -1
					return
				}
				took = time.Since(start)
			})
			elapsed <- took
		}()
		select {
		case took := <-elapsed:
			if took < 0 {
				return fmt.Errorf("mouse down was not sent")
			}
			if took > 200*time.Millisecond {
				return fmt.Errorf("mouse down blocked for %s", took)
			}
			fmt.Printf("CLICK %s\n", took)
			return nil
		case <-time.After(1500 * time.Millisecond):
			return fmt.Errorf("mouse down did not return")
		}
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func sendMouseDown(view objc.ID) error {
	wnd := view.Send(objc.RegisterName("window"))
	if wnd == 0 {
		return fmt.Errorf("view has no window")
	}
	rect := boundsOf(view)
	ev := objc.ID(objc.GetClass("NSEvent")).Send(
		objc.RegisterName("mouseEventWithType:location:modifierFlags:timestamp:windowNumber:context:eventNumber:clickCount:pressure:"),
		1,
		nsPoint{X: rect.Size.Width / 2, Y: rect.Size.Height / 2},
		0,
		float64(0),
		int(wnd.Send(objc.RegisterName("windowNumber"))),
		objc.ID(0),
		1,
		1,
		float32(1),
	)
	if ev == 0 {
		return fmt.Errorf("no mouse event")
	}
	app := objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
	app.Send(selSendEvent, ev)
	return nil
}
