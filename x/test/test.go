// Package test is helpers for tests: process globals, closers, iterators, and readers.
//
// Stdout, Stderr, Slog, DiscardSlog, and RestoreSlog mutate process globals.
// Do not call t.Parallel in those tests.
//
// Database open and migrate stay on x/db. CLI is x/cmd, not Cobra.
package test

import (
	"io"
	"iter"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ErrorReader is an io.Reader whose Read always returns Err.
type ErrorReader struct{ Err error }

func (r ErrorReader) Read([]byte) (int, error) { return 0, r.Err }

// CloseOnCleanup registers c.Close when the test ends.
func CloseOnCleanup(tb testing.TB, c io.Closer) {
	tb.Helper()
	tb.Cleanup(func() {
		err := c.Close()
		assert.NoErrorf(tb, err, "close: %v", err)
	})
}

// Collect drains seq and fatals on the first error.
func Collect[T any](tb testing.TB, seq iter.Seq2[T, error]) []T {
	tb.Helper()
	var out []T
	for v, err := range seq {
		require.NoError(tb, err)
		out = append(out, v)
	}
	return out
}

// Stdout runs fn with os.Stdout swapped to a pipe and returns what was written.
func Stdout(tb testing.TB, fn func()) string {
	tb.Helper()
	return capture(tb, &os.Stdout, fn)
}

// Stderr runs fn with os.Stderr swapped to a pipe and returns what was written.
func Stderr(tb testing.TB, fn func()) string {
	tb.Helper()
	return capture(tb, &os.Stderr, fn)
}

func capture(tb testing.TB, dest **os.File, fn func()) string {
	tb.Helper()
	r, w, err := os.Pipe()
	require.NoError(tb, err)
	old := *dest
	*dest = w
	closed := false
	closeW := func() error {
		if closed {
			return nil
		}
		closed = true
		*dest = old
		return w.Close()
	}
	tb.Cleanup(func() {
		err := closeW()
		assert.NoErrorf(tb, err, "close pipe: %v", err)
	})

	type result struct {
		s   string
		err error
	}
	done := make(chan result, 1)
	go func() {
		b, err := io.ReadAll(r)
		done <- result{string(b), err}
	}()

	fn()
	require.NoError(tb, closeW())
	got := <-done
	require.NoError(tb, got.err)
	require.NoError(tb, r.Close())
	return got.s
}

// RestoreSlog snapshots slog.Default and puts it back when the test ends.
func RestoreSlog(tb testing.TB) {
	tb.Helper()
	prev := slog.Default()
	tb.Cleanup(func() { slog.SetDefault(prev) })
}

// Slog installs h as the default logger and restores the previous default
// when the test ends.
func Slog(tb testing.TB, h slog.Handler) {
	tb.Helper()
	RestoreSlog(tb)
	slog.SetDefault(slog.New(h))
}

// DiscardSlog installs slog.DiscardHandler for the rest of the test.
func DiscardSlog(tb testing.TB) {
	tb.Helper()
	Slog(tb, slog.DiscardHandler)
}

// Skip marks tb skipped. Tests call this instead of tb.Skip.
func Skip(tb testing.TB, args ...any) {
	tb.Helper()
	tb.Skip(args...)
}

// Skipf marks tb skipped with a formatted reason.
func Skipf(tb testing.TB, format string, args ...any) {
	tb.Helper()
	tb.Skipf(format, args...)
}

// Need returns the absolute path of name on PATH, or skips the test.
func Need(tb testing.TB, name string) string {
	tb.Helper()
	p, err := execdriver.Which(tb.Context(), name)
	if err != nil {
		Skipf(tb, "missing %s", name)
	}
	abs, err := filepath.Abs(p)
	require.NoError(tb, err)
	return abs
}
