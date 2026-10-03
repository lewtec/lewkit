package app

import (
	"net/http"
	"os"
	"runtime"
	"testing"

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

func TestLoopbackLinkUsesIPv4(t *testing.T) {
	require.Equal(t, "http://127.0.0.1:9/?token=abc", loopbackLink("http://[::1]:9", "abc"))
}

func TestDesktopIsNotNativeHost(t *testing.T) {
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		t.Skip()
	}
	require.False(t, nativeHost(GUI(nil)))
	require.False(t, nativeHost(Web(nil)))
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

func TestRunPanicsWithoutStamp(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = App{NoUI: true}.Run(t.Context())
}
