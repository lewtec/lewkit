package term

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/lewtec/lewkit/x/release"
)

// Socket is the default unix socket for the ubersession.
// It is $XDG_RUNTIME_DIR/<name>/term.sock, or a directory under the temp dir
// named with the user id when XDG_RUNTIME_DIR is unset.
func Socket() (string, error) {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = filepath.Join(os.TempDir(), fmt.Sprintf("%s-%d", release.Name(), os.Getuid()))
	}
	return filepath.Join(dir, release.Name(), "term.sock"), nil
}

// Listen serves a new ubersession on the unix socket at path.
// A socket that accepts a connection is left alone and Listen returns [ErrRunning].
// A stale socket file is removed. Listen returns when ctx is cancelled.
// It also returns after a client has connected and then the last client
// disconnects while no panes remain.
func Listen(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("%w: %s", ErrSocket, "empty socket path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("%w: %w", ErrSocket, err)
	}
	if err := reclaim(ctx, path); err != nil {
		return err
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSocket, err)
	}
	defer listener.Close()
	defer os.Remove(path)
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("%w: %w", ErrSocket, err)
	}
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	session, err := Open(ctx)
	if err != nil {
		return err
	}
	defer session.Close()
	session.stopWhenIdle(stop)
	info, err := session.Info(ctx)
	if err != nil {
		return err
	}
	slog.Info("term listening", "socket", path, "session", info.ID)
	return session.Serve(ctx, listener)
}

// Connect dials the ubersession at path.
// When nothing is listening, it calls start and dials until the socket accepts.
// start must return after the server process is spawned. It must not wait for
// that process to exit. A running server does not call start.
// ErrRunning from start is ignored when the socket accepts, so two clients
// can race to spawn. The caller closes the returned client.
func Connect(ctx context.Context, path string, start func(context.Context) error) (*Client, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if path == "" {
		return nil, fmt.Errorf("%w: %s", ErrSocket, "empty socket path")
	}
	client, err := Dial(ctx, path)
	if err == nil {
		return client, nil
	}
	if !errors.Is(err, ErrSocket) {
		return nil, err
	}
	if start == nil {
		return nil, err
	}
	startErr := start(ctx)
	client, err = Dial(ctx, path)
	if err == nil {
		return client, nil
	}
	if startErr != nil && !errors.Is(startErr, ErrRunning) {
		return nil, startErr
	}
	return waitDial(ctx, path)
}

func waitDial(ctx context.Context, path string) (*Client, error) {
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	var last error
	for {
		client, err := Dial(ctx, path)
		if err == nil {
			return client, nil
		}
		last = err
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			return nil, last
		case <-ticker.C:
		}
	}
}

func reclaim(ctx context.Context, path string) error {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("%w: %w", ErrSocket, err)
	}
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "unix", path)
	if err == nil {
		conn.Close()
		return fmt.Errorf("%w: %s", ErrRunning, path)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %w", ErrSocket, err)
	}
	return nil
}
