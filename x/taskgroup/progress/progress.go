// Package progress is a bubbletea view of a taskgroup Session.
//
// Run polls Session.List until work finishes and draws a nom-style
// tree (├ └ │, status glyphs). The TUI starts on the first Go or
// LineWriter, not when Run is entered. The last frame is empty: the
// view quits only after Wait and List is empty. A missing terminal,
// TERM=dumb, CI, and NO_COLOR skip the TUI and Wait. Android links
// the wait path only.
package progress

import (
	"context"
	"os"
	"strings"

	"github.com/lewtec/lewkit/x/taskgroup"
	"golang.org/x/term"
)

const defaultMaxRows = 16

// Interactive is false for TERM=dumb, NO_COLOR, CI, or when the view
// cannot take a terminal. /dev/null is a character device and is not a
// terminal. LEWKIT_FORCE_TUI=1 draws even when TERM=dumb, NO_COLOR, or
// CI is set, and still requires a real terminal.
func Interactive() bool {
	forced := os.Getenv("LEWKIT_FORCE_TUI") != ""
	if !forced && (os.Getenv("TERM") == "dumb" || os.Getenv("NO_COLOR") != "" || os.Getenv("CI") != "") {
		return false
	}
	return terminalView(os.Stdin, os.Stderr)
}

// terminalView reports that out is a terminal and input can be read.
// Bubbletea reads stdin, and opens /dev/tty when stdin is not a terminal.
func terminalView(in, out *os.File) bool {
	if !isTerminal(out) {
		return false
	}
	if isTerminal(in) {
		return true
	}
	return ttyAvailable()
}

func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

func ttyAvailable() bool {
	file, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	_ = file.Close()
	return true
}

func ttyUnavailable(err error) bool {
	return err != nil && strings.Contains(err.Error(), "could not open TTY")
}

type runKey struct{}

// Run runs work, then waits for the session. On a tty it shows live bars
// from Session.List while that happens. tea.NewProgram runs at most once,
// on the first scheduled task or LineWriter. A nested Run uses the
// session already on ctx and does not start a second view.
func Run(s *taskgroup.Session, ctx context.Context, work func(context.Context) error) error {
	if ctx.Value(runKey{}) != nil {
		return work(ctx)
	}
	if s == nil {
		return work(ctx)
	}
	ctx = context.WithValue(ctx, runKey{}, true)
	if !Interactive() {
		err := work(ctx)
		if werr := s.Wait(); err == nil {
			err = werr
		}
		return err
	}
	return runTea(s, ctx, work)
}
