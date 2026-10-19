package daemon_test

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/daemon"
	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if os.Getenv("LEWKIT_DAEMON_CHILD") == "1" {
		daemonChild()
		return
	}
	os.Exit(m.Run())
}

func daemonChild() {
	path := os.Getenv("LEWKIT_DAEMON_FILE")
	text := strconv.Itoa(os.Getpid())
	if sid := childSession(); sid >= 0 {
		text += " " + strconv.Itoa(sid)
	}
	_ = os.WriteFile(path, []byte(text), 0o600)
	if os.Getenv("LEWKIT_DAEMON_SLEEP") == "1" {
		time.Sleep(5 * time.Second)
	}
}

func TestSelfReturnsBeforeTheChildExits(t *testing.T) {
	file := filepath.Join(t.TempDir(), "child")
	t.Setenv("LEWKIT_DAEMON_CHILD", "1")
	t.Setenv("LEWKIT_DAEMON_FILE", file)
	t.Setenv("LEWKIT_DAEMON_SLEEP", "1")

	started := time.Now()
	require.NoError(t, daemon.Self(t.Context()))
	require.Less(t, time.Since(started), time.Second)

	var pid, sid int
	require.Eventually(t, func() bool {
		blob, err := os.ReadFile(file)
		if err != nil {
			return false
		}
		fields := strings.Fields(string(blob))
		if len(fields) == 0 {
			return false
		}
		var conv error
		pid, conv = strconv.Atoi(fields[0])
		if conv != nil || pid <= 0 {
			return false
		}
		if len(fields) == 1 {
			sid = -1
			return true
		}
		sid, conv = strconv.Atoi(fields[1])
		return conv == nil
	}, 3*time.Second, 20*time.Millisecond)
	t.Cleanup(func() {
		proc, err := os.FindProcess(pid)
		if err == nil {
			_ = proc.Kill()
		}
	})
	if sid >= 0 {
		require.Equal(t, pid, sid)
	}
}

func TestSpawnRejectsABadStart(t *testing.T) {
	require.ErrorIs(t, daemon.Spawn(nil, "true"), errs.ErrNilContext)
	require.ErrorIs(t, daemon.Self(nil), errs.ErrNilContext)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, daemon.Spawn(ctx, "true"), context.Canceled)
	require.ErrorIs(t, daemon.Self(ctx), context.Canceled)

	err := daemon.Spawn(t.Context(), "")
	require.ErrorIs(t, err, daemon.ErrSpawn)

	err = daemon.Spawn(t.Context(), "lewkit-daemon-missing")
	require.ErrorIs(t, err, daemon.ErrSpawn)
}
