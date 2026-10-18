package term

import (
	"bufio"
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/term/schema"
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

	_, err = client.Call(ctx, Method("nope"), nil)
	require.ErrorIs(t, err, ErrMethod)

	raw := dialRaw(t, path)
	putFrame(t, raw, &schema.Envelope{Version: 2, Id: 1, Method: string(MethodSessionGet)})
	bad := takeFrame(t, raw)
	require.ErrorIs(t, failureFromSchema(bad.GetError()), ErrVersion)

	unknown := dialRaw(t, path)
	putFrame(t, unknown, &schema.Envelope{Version: ProtocolVersion, Id: 2, Method: "nope"})
	missing := takeFrame(t, unknown)
	require.ErrorIs(t, failureFromSchema(missing.GetError()), ErrMethod)

	info, err := client.Session(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, info.ID)
	listed, err := client.Panes(ctx)
	require.NoError(t, err)
	assert.Empty(t, listed)

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

func TestSchemaRoundTrip(t *testing.T) {
	pane := Pane{
		PaneInfo: PaneInfo{
			ID: "p", Title: "t", Command: []string{"/bin/cat"},
			Alive: false, Exit: -1, Rows: 1, Cols: 2, Revision: 3,
			LastWriter: "c", CursorCol: 1,
		},
		Cells: []Cell{{Rune: "a", FG: 2, Bold: true}, {Wide: true}},
	}
	assert.Equal(t, pane, paneFromSchema(paneToSchema(pane)))
	req := WriteRequest{ID: "p", Data: "\x03", Client: "c"}
	assert.Equal(t, req, writeFromSchema(writeToSchema(req)))
	watch := WatchRequest{Panes: []string{"p"}}
	assert.Equal(t, watch, watchFromSchema(watchToSchema(watch)))
	assert.Equal(t, WatchRequest{All: true}, watchFromSchema(watchToSchema(WatchRequest{All: true})))
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

func putFrame(t *testing.T, conn net.Conn, msg *schema.Envelope) {
	t.Helper()
	require.NoError(t, writeFrame(bufio.NewWriter(conn), msg))
}

func takeFrame(t *testing.T, conn net.Conn) *schema.Envelope {
	t.Helper()
	msg, err := readFrame(bufio.NewReader(conn))
	require.NoError(t, err)
	return msg
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
