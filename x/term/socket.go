package term

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"

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
	session, err := Open(ctx)
	if err != nil {
		return err
	}
	defer session.Close()
	info, err := session.Info(ctx)
	if err != nil {
		return err
	}
	slog.Info("term listening", "socket", path, "session", info.ID)
	return session.Serve(ctx, listener)
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
