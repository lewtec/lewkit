// Package progress is a bubbletea view of a taskgroup Session.
//
// Run polls Session.List until work finishes and draws a nom-style
// tree (├ └ │, status glyphs). The last frame is empty: the view
// quits only after Wait and List is empty. Non-tty, TERM=dumb, CI,
// and NO_COLOR skip the TUI and Wait. LEWKIT_FORCE_TUI=1 forces the TUI.
package progress

import (
	"context"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/lewtec/lewkit/x/taskgroup"
)

const defaultMaxRows = 16

// Interactive is false for TERM=dumb, NO_COLOR, CI, or a non-tty stdout/stderr.
func Interactive() bool {
	if os.Getenv("LEWKIT_FORCE_TUI") != "" {
		return true
	}
	if os.Getenv("TERM") == "dumb" || os.Getenv("NO_COLOR") != "" || os.Getenv("CI") != "" {
		return false
	}
	return isCharDevice(os.Stdout) && isCharDevice(os.Stderr)
}

func isCharDevice(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// Run runs work, then waits for the session. On a tty it shows live bars
// from Session.List while that happens.
func Run(s *taskgroup.Session, ctx context.Context, work func(context.Context) error) error {
	if s == nil {
		return work(ctx)
	}
	if !Interactive() {
		err := work(ctx)
		if werr := s.Wait(); err == nil {
			err = werr
		}
		return err
	}
	return runTea(s, ctx, work)
}

type stopKey struct{}

// WithStop records cancel so the TUI can abort the session on ctrl+c.
func WithStop(ctx context.Context, stop context.CancelFunc) context.Context {
	return context.WithValue(ctx, stopKey{}, stop)
}

func runTea(s *taskgroup.Session, ctx context.Context, work func(context.Context) error) error {
	m := newModel(s)
	if stop, ok := ctx.Value(stopKey{}).(context.CancelFunc); ok {
		m.cancel = stop
	}
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))

	errc := make(chan error, 1)
	go func() {
		err := work(ctx)
		if werr := s.Wait(); err == nil {
			err = werr
		}
		errc <- err
		p.Send(doneMsg{})
	}()

	_, uiErr := p.Run()
	if m.cancel != nil {
		m.cancel()
	}
	err := <-errc
	if uiErr != nil && err == nil {
		return uiErr
	}
	return err
}

type tickMsg time.Time

type doneMsg struct{}

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
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
		}
		return m, nil
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
