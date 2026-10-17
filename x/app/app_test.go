package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestAnnounceReadyWritesFile(t *testing.T) {
	path := t.TempDir() + "/ready.url"
	t.Setenv("ELETROCROMO_READY_FILE", path)
	announceReady("http://127.0.0.1:9/?token=abc")
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:9/?token=abc\n", string(body))
}

func TestAnnounceSurfaceWritesEmptyReadyLineInAppMode(t *testing.T) {
	path := t.TempDir() + "/ready.url"
	t.Setenv("ELETROCROMO_READY_FILE", path)
	t.Setenv("ELETROCROMO_NO_UI", "1")
	announceSurface()
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "\n", string(body))
}

func TestLoopbackLinkUsesIPv4(t *testing.T) {
	require.Equal(t, "http://127.0.0.1:9/?token=abc", loopbackLink("http://[::1]:9", "abc"))
}

func TestDesktopIsNotNativeHost(t *testing.T) {
	if runtime.GOOS == "android" || runtime.GOOS == "darwin" || runtime.GOOS == "ios" {
		test.Skip(t)
	}
	require.False(t, nativeHost(GUI(nil)))
	require.False(t, nativeHost(Web(nil)))
}

func TestAndroidHostKeepsEveryWindow(t *testing.T) {
	require.True(t, keepNative("android", GUI(nil), true))
	require.True(t, keepNative("android", Web(nil), true))
}

func TestAndroidWithoutHostPublishesLoopback(t *testing.T) {
	web := Web(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	require.False(t, keepNative("android", GUI(nil), false))
	require.False(t, keepNative("android", web, false))
	win, err := loopbackWindow(web, false)
	require.NoError(t, err)
	hosted, ok := win.(webWindow)
	require.True(t, ok)
	require.True(t, hosted.hosted)
}

func TestDarwinKeepsGUIOnly(t *testing.T) {
	require.True(t, keepNative("darwin", GUI(nil), false))
	require.False(t, keepNative("darwin", Web(nil), true))
}

func TestIOSKeepsGUIOnly(t *testing.T) {
	require.True(t, keepNative("ios", GUI(nil), false))
	require.False(t, keepNative("ios", Web(nil), true))
}

func TestLoopbackKeepsAndroidGUI(t *testing.T) {
	win, err := loopbackWindow(GUI(nil), true)
	require.NoError(t, err)
	_, ok := win.(guiWindow)
	require.True(t, ok)
}

func TestLoopbackRejectsDesktopGUI(t *testing.T) {
	_, err := loopbackWindow(GUI(nil), false)
	require.EqualError(t, err, "loopback host needs a web handler")
}

func TestLoopbackLeavesAndroidWeb(t *testing.T) {
	win, err := loopbackWindow(Web(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})), true)
	require.NoError(t, err)
	web, ok := win.(webWindow)
	require.True(t, ok)
	require.False(t, web.hosted)
}

func TestLoopbackHostsWeb(t *testing.T) {
	win, err := loopbackWindow(Web(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})), false)
	require.NoError(t, err)
	web, ok := win.(webWindow)
	require.True(t, ok)
	require.True(t, web.hosted)
}

func TestMissingWebViewDoesNotLoopback(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- webWindow{handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})}.open(ctx, "Demo", 100, 100, "")
	}()
	select {
	case err := <-done:
		require.ErrorIs(t, err, driver.ErrUnavailable)
	case <-time.After(2 * time.Second):
		// A host with a webview blocks in the view. Cancel closes it.
		// Open can return nil once the view is up, or context.Canceled
		// when the cancel lands inside Open. Either means it stopped.
		cancel()
		select {
		case err := <-done:
			if err != nil && !errors.Is(err, context.Canceled) {
				require.NoError(t, err)
			}
		case <-time.After(2 * time.Second):
			require.FailNow(t, "webview open did not return after cancel")
		}
	}
}

func TestRunPanicsWithoutStamp(t *testing.T) {
	defer func() {
		if recover() == nil {
			require.FailNow(t, fmt.Sprint("expected panic"))
		}
	}()
	_ = App{NoUI: true}.Run(t.Context())
}
