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
)

type closeCount struct{ n int }

func (c *closeCount) Close() error {
	c.n++
	return nil
}

func TestClose(t *testing.T) {
	var c closeCount
	t.Run("inner", func(t *testing.T) {
		Close(t, &c)
	})
	if c.n != 1 {
		t.Fatalf("closed %d times, want 1", c.n)
	}
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
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("Collect() = %v, want [1 2]", got)
	}
}

func TestStdout(t *testing.T) {
	got := Stdout(t, func() {
		_, err := io.WriteString(os.Stdout, "hello")
		if err != nil {
			t.Fatal(err)
		}
	})
	if got != "hello" {
		t.Fatalf("Stdout() = %q, want %q", got, "hello")
	}
	if os.Stdout == nil {
		t.Fatal("os.Stdout is nil after Stdout")
	}
}

func TestStdoutLarge(t *testing.T) {
	want := strings.Repeat("x", 1<<20)
	got := Stdout(t, func() {
		_, err := io.WriteString(os.Stdout, want)
		if err != nil {
			t.Fatal(err)
		}
	})
	if got != want {
		t.Fatalf("Stdout() len = %d, want %d", len(got), len(want))
	}
}

func TestStderr(t *testing.T) {
	got := Stderr(t, func() {
		_, err := io.WriteString(os.Stderr, "err")
		if err != nil {
			t.Fatal(err)
		}
	})
	if got != "err" {
		t.Fatalf("Stderr() = %q, want %q", got, "err")
	}
}

func TestSlogRestores(t *testing.T) {
	prev := slog.Default()
	t.Run("inner", func(t *testing.T) {
		DiscardSlog(t)
		if slog.Default() == prev {
			t.Fatal("DiscardSlog left the previous default")
		}
	})
	if slog.Default() != prev {
		t.Fatal("slog.Default was not restored")
	}
}

func TestSlogCaptures(t *testing.T) {
	var buf bytes.Buffer
	Slog(t, slog.NewTextHandler(&buf, nil))
	slog.Info("ping")
	if !strings.Contains(buf.String(), "ping") {
		t.Fatalf("log = %q, want ping", buf.String())
	}
}

func TestNeedFindsSh(t *testing.T) {
	p := Need(t, "sh")
	if !filepath.IsAbs(p) {
		t.Fatalf("Need(sh) = %q, want absolute", p)
	}
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
		t.Fatal("Need should have skipped")
	})
	if inner == nil || !inner.Skipped() {
		t.Fatal("subtest did not skip")
	}
}
