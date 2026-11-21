package test

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type closeCount struct{ n int }

func (c *closeCount) Close() error {
	c.n++
	return nil
}

func TestErrorReader(t *testing.T) {
	r := ErrorReader{Err: io.ErrUnexpectedEOF}
	n, err := r.Read(make([]byte, 8))
	require.Equal(t, 0, n)
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
}

func TestCloseOnCleanup(t *testing.T) {
	var c closeCount
	t.Run("inner", func(t *testing.T) {
		CloseOnCleanup(t, &c)
	})
	require.Equal(t, 1, c.n)
}

func TestCollect(t *testing.T) {
	seq := func(yield func(int, error) bool) {
		if !yield(1, nil) {
			return
		}
		if !yield(2, nil) {
			return
		}
	}
	got := Collect(t, seq)
	require.Equal(t, []int{1, 2}, got)
}

func TestStdout(t *testing.T) {
	got := Stdout(t, func() {
		_, err := io.WriteString(os.Stdout, "hello")
		require.NoError(t, err)
	})
	require.Equal(t, "hello", got)
	require.NotNil(t, os.Stdout)
}

func TestStdoutLarge(t *testing.T) {
	want := strings.Repeat("x", 1<<20)
	got := Stdout(t, func() {
		_, err := io.WriteString(os.Stdout, want)
		require.NoError(t, err)
	})
	require.Equal(t, want, got)
}

func TestStderr(t *testing.T) {
	got := Stderr(t, func() {
		_, err := io.WriteString(os.Stderr, "err")
		require.NoError(t, err)
	})
	require.Equal(t, "err", got)
}

func TestSlogRestores(t *testing.T) {
	prev := slog.Default()
	t.Run("inner", func(t *testing.T) {
		DiscardSlog(t)
		require.NotEqual(t, prev, slog.Default())
	})
	require.Equal(t, prev, slog.Default())
}

func TestSlogCaptures(t *testing.T) {
	var buf bytes.Buffer
	Slog(t, slog.NewTextHandler(&buf, nil))
	slog.Info("ping")
	require.Contains(t, buf.String(), "ping")
}

func TestNeedFindsSh(t *testing.T) {
	p := Need(t, "sh")
	require.True(t, filepath.IsAbs(p))
}

func TestNeedSkipsMissing(t *testing.T) {
	const missing = "lewkit-no-such-bin"
	if _, err := exec.LookPath(missing); err == nil {
		t.Skipf("%s is on PATH", missing)
	}
	var inner *testing.T
	t.Run("missing", func(t *testing.T) {
		inner = t
		Need(t, missing)
		require.FailNow(t, "Need should have skipped")
	})
	require.NotNil(t, inner)
	require.True(t, inner.Skipped())
}
