package app

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
)

func TestMissingWebViewDoesNotLoopback(t *testing.T) {
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("LEWKIT_ENABLE_MEMORY_DRIVER", "")
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	err := webWindow{handler: http.NotFoundHandler()}.open(ctx, "t", 800, 600, t.TempDir())
	require.ErrorIs(t, err, driver.ErrUnavailable)
}

func TestMissingWebViewLoopbackNeedsMemoryEnv(t *testing.T) {
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("LEWKIT_ENABLE_MEMORY_DRIVER", "1")
	path := filepath.Join(t.TempDir(), "ready")
	t.Setenv("ELETROCROMO_READY_FILE", path)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- webWindow{handler: http.NotFoundHandler()}.open(ctx, "t", 800, 600, t.TempDir())
	}()
	require.Eventually(t, func() bool {
		select {
		case err := <-done:
			require.NoError(t, err)
			return false
		default:
		}
		_, err := os.Stat(path)
		return err == nil
	}, 5*time.Second, 10*time.Millisecond)
	cancel()
	require.Eventually(t, func() bool {
		select {
		case err := <-done:
			require.NoError(t, err)
			return true
		default:
			return false
		}
	}, 5*time.Second, 10*time.Millisecond)
}

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

func TestRunPanicsWithoutStamp(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = App{NoUI: true}.Run(t.Context())
}
