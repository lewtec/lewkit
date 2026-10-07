// Package progress is a bubbletea view of a taskgroup Session.
//
// Run polls Session.List until work finishes and draws a nom-style
// tree (├ └ │, status glyphs). The TUI starts on the first Go or
// LineWriter, not when Run is entered. The last frame is empty: the
// view quits only after Wait and List is empty. A missing terminal,
// TERM=dumb, CI, and NO_COLOR skip the TUI and Wait.
package progress

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
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
	if ctx != nil && ctx.Value(runKey{}) != nil {
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

func runTea(s *taskgroup.Session, ctx context.Context, work func(context.Context) error) (err error) {
	var ui teaUI
	s.SetOnSchedule(sync.OnceFunc(ui.start(s, ctx)))
	defer s.SetOnSchedule(nil)
	defer func() {
		if uiErr := ui.stop(s); uiErr != nil && err == nil {
			err = uiErr
		}
	}()

	err = work(ctx)
	if werr := s.Wait(); err == nil {
		err = werr
	}
	return err
}

type teaUI struct {
	p       *tea.Program
	done    chan struct{}
	err     error
	restore func()
}

func (u *teaUI) start(s *taskgroup.Session, ctx context.Context) func() {
	return func() {
		m := newModel(s)
		u.p = tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
			if _, ok := msg.(tea.InterruptMsg); ok {
				return cancelMsg{}
			}
			return msg
		}))
		s.SetLinePrint(u.print)
		u.restore = hijackSlog(ctx, u.print)
		u.done = make(chan struct{})
		go func() {
			defer close(u.done)
			_, u.err = u.p.Run()
			if ttyUnavailable(u.err) {
				u.err = nil
			}
			u.restoreLogs(s)
			if u.err != nil {
				s.Cancel(u.err)
			}
		}()
	}
}

// print uses Program.Send so a log after Run returns is a no-op.
// Program.Printf sends on p.msgs with no ctx.Done and blocks forever
// once the event loop has exited.
func (u *teaUI) print(s string) {
	if u.p == nil {
		return
	}
	u.p.Send(printLineMsg(s))
}

func (u *teaUI) restoreLogs(s *taskgroup.Session) {
	if u.restore != nil {
		u.restore()
	}
	s.SetLinePrint(nil)
}

func (u *teaUI) stop(s *taskgroup.Session) error {
	if u.p == nil {
		return nil
	}
	u.restoreLogs(s)
	for _, line := range s.TakeLiveLines() {
		_, _ = os.Stderr.WriteString(line + "\n")
	}
	select {
	case <-u.done:
	default:
		u.p.Send(doneMsg{})
		<-u.done
	}
	return u.err
}

type tickMsg time.Time

type doneMsg struct{}

type cancelMsg struct{}

type printLineMsg string

func (m model) Init() tea.Cmd {
	return m.tick()
}

func (m model) tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *model) refresh() {
	if m.session != nil {
		m.sync(m.session.List(m.max))
		m.live = m.session.LiveLines()
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			m.requestStop()
			return m, nil
		}
	case cancelMsg:
		m.requestStop()
		return m, nil
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
		}
		return m, nil
	case printLineMsg:
		return m, tea.Printf("%s", string(msg))
	case doneMsg:
		m.done = true
		m.refresh()
		if m.shouldQuit() {
			return m, tea.Quit
		}
		return m, m.tick()
	case tickMsg:
		m.refresh()
		if m.shouldQuit() {
			return m, tea.Quit
		}
		return m, m.tick()
	}
	return m, nil
}
