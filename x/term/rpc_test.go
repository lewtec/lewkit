package term

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRPCVersionMethodAndWatch(t *testing.T) {
	if _, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0); err != nil {
		test.Skipf(t, "pty: %v", err)
	}
	ctx := t.Context()
	path, session := serveRPC(t, ctx)

	client, err := Dial(ctx, path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	err = client.Call(ctx, Method("nope"), nil, nil)
	require.ErrorIs(t, err, ErrMethod)

	raw := dialRaw(t, path)
	require.NoError(t, json.NewEncoder(raw).Encode(map[string]any{
		"v": 2, "id": 1, "method": "session.get",
	}))
	var bad wireIn
	require.NoError(t, json.NewDecoder(raw).Decode(&bad))
	require.NotNil(t, bad.Error)
	require.ErrorIs(t, bad.Error, ErrVersion)

	unknown := dialRaw(t, path)
	require.NoError(t, json.NewEncoder(unknown).Encode(map[string]any{
		"v": ProtocolVersion, "id": 2, "method": "nope",
	}))
	var missing wireIn
	require.NoError(t, json.NewDecoder(unknown).Decode(&missing))
	require.NotNil(t, missing.Error)
	require.ErrorIs(t, missing.Error, ErrMethod)

	first, err := client.Create(ctx, CreateRequest{Command: []string{"/bin/cat"}, Rows: 24, Cols: 80})
	require.NoError(t, err)

	only, err := Dial(ctx, path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = only.Close() })
	require.NoError(t, only.Watch(ctx, WatchRequest{Panes: []string{first.ID}}))
	drainEvents(t, only)

	all, err := Dial(ctx, path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = all.Close() })
	require.NoError(t, all.Watch(ctx, WatchRequest{All: true}))
	drainEvents(t, all)

	second, err := session.Create(ctx, CreateRequest{Command: []string{"/bin/cat"}, Rows: 24, Cols: 80})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		wait, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()
		ev, err := all.Next(wait)
		if err != nil {
			return false
		}
		return ev.Notice == NoticePane && ev.Pane != nil && ev.Pane.ID == second.ID ||
			ev.Notice == NoticeSession && ev.Session != nil && contains(ev.Session.Panes, second.ID)
	}, 2*time.Second, 10*time.Millisecond)

	quiet, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	_, err = only.Next(quiet)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	require.NoError(t, client.Write(ctx, WriteRequest{
		ID: first.ID, Data: "hi\n", Rows: 24, Cols: 80, Client: "rpc",
	}))
	require.Eventually(t, func() bool {
		pane, err := client.Pane(ctx, first.ID)
		if err != nil {
			return false
		}
		text := pane.Line(0) + pane.Line(1)
		return strings.Contains(text, "hi")
	}, 2*time.Second, 20*time.Millisecond)
}

func TestListenStaleSocket(t *testing.T) {
	path := filepath.Join(t.TempDir(), "term.sock")
	ctx, cancel := context.WithCancel(t.Context())
	errCh := make(chan error, 1)
	go func() { errCh <- Listen(ctx, path) }()
	require.Eventually(t, func() bool {
		client, err := Dial(t.Context(), path)
		if err != nil {
			return false
		}
		_ = client.Close()
		return true
	}, 2*time.Second, 10*time.Millisecond)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	err = Listen(t.Context(), path)
	require.ErrorIs(t, err, ErrRunning)

	cancel()
	require.NoError(t, <-errCh)

	ctx, cancel = context.WithCancel(t.Context())
	go func() { errCh <- Listen(ctx, path) }()
	require.Eventually(t, func() bool {
		client, err := Dial(t.Context(), path)
		if err != nil {
			return false
		}
		_ = client.Close()
		return true
	}, 2*time.Second, 10*time.Millisecond)
	cancel()
	require.NoError(t, <-errCh)
}

func serveRPC(t *testing.T, ctx context.Context) (string, *Session) {
	t.Helper()
	session, err := Open(ctx)
	require.NoError(t, err)
	t.Cleanup(session.Close)
	path := filepath.Join(t.TempDir(), "term.sock")
	listener, err := net.Listen("unix", path)
	require.NoError(t, err)
	go func() { _ = session.Serve(ctx, listener) }()
	t.Cleanup(func() { _ = listener.Close() })
	return path, session
}

func dialRaw(t *testing.T, path string) net.Conn {
	t.Helper()
	conn, err := net.Dial("unix", path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func drainEvents(t *testing.T, client *Client) {
	t.Helper()
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		wait, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
		_, err := client.Next(wait)
		cancel()
		if err != nil {
			return
		}
	}
}
