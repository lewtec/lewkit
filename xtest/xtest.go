// Package xtest is process-wide test seams: stdout, stderr, slog, and PATH.
//
// Stdout, Stderr, Slog, DiscardSlog, and RestoreSlog mutate process globals.
// Do not call t.Parallel in those tests.
//
// Database open and migrate stay on x/db. CLI is x/cmd, not Cobra.
package xtest

import (
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

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
	if err != nil {
		tb.Fatal(err)
	}
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
		if err := closeW(); err != nil {
			tb.Errorf("close pipe: %v", err)
		}
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
	if err := closeW(); err != nil {
		tb.Fatal(err)
	}
	got := <-done
	if got.err != nil {
		tb.Fatal(got.err)
	}
	if err := r.Close(); err != nil {
		tb.Fatal(err)
	}
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

// Need returns the absolute path of name on PATH, or skips the test.
func Need(tb testing.TB, name string) string {
	tb.Helper()
	p, err := exec.LookPath(name)
	if err != nil {
		tb.Skipf("missing %s", name)
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		tb.Fatal(err)
	}
	return abs
}
